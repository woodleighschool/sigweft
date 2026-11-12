package server

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/woodleighschool/sigweft/internal/account"
	"github.com/woodleighschool/sigweft/internal/config"
	"github.com/woodleighschool/sigweft/internal/signature"
	"golang.org/x/oauth2"
)

func TestSSOGateAndEditableSignature(t *testing.T) {
	for _, email := range []string{"jo@example.org", ""} {
		t.Run("email="+email, func(t *testing.T) {
			app := httptest.NewUnstartedServer(nil)
			appURL := "http://" + app.Listener.Addr().String()
			issuer := oidcIssuer(t, email, appURL+"/api/auth/sso/callback")
			logger := slog.New(slog.DiscardHandler)
			auth, err := account.New(t.Context(), config.Runtime{
				OIDCIssuerURL: issuer.URL, OIDCClientID: "sigweft", OIDCClientSecret: "synthetic-secret",
				OIDCRedirectURL: appURL + "/api/auth/sso/callback",
			}, logger)
			if err != nil {
				t.Fatal(err)
			}
			renderer, err := signature.Load("../../examples/studio/sigweft.yaml")
			if err != nil {
				t.Fatal(err)
			}
			app.Config.Handler, _ = New(renderer, fstest.MapFS{"index.html": {Data: []byte("app shell")}}, logger, "test", auth)
			app.Start()
			t.Cleanup(app.Close)
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			request := func(method, target, body, origin string, wantStatus int) (http.Header, []byte) {
				t.Helper()
				req, err := http.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "application/json")
				if origin != "" {
					req.Header.Set("Origin", origin)
				}
				res, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = res.Body.Close() }()
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				if res.StatusCode != wantStatus {
					t.Fatalf("%s %s: status=%d body=%s", method, target, res.StatusCode, data)
				}
				return res.Header, data
			}
			getSession := func() account.Session {
				t.Helper()
				h, data := request(http.MethodGet, appURL+"/api/session", "", "", 200)
				if h.Get("Cache-Control") != "no-store" {
					t.Fatal("session can be cached")
				}
				var session account.Session
				if err := json.Unmarshal(data, &session); err != nil {
					t.Fatal(err)
				}
				return session
			}
			if session := getSession(); !session.SSOEnabled || session.User != nil {
				t.Fatalf("anonymous session=%+v", session)
			}
			request(http.MethodGet, appURL+"/api/form", "", "", 401)
			request(http.MethodPost, appURL+"/api/render", `{"values":{}}`, "", 401)
			request(http.MethodGet, appURL+"/healthz", "", "", 200)
			request(http.MethodGet, appURL+"/login", "", "", 200)

			// A failed callback must not create a session or grant access.
			request(http.MethodGet, appURL+"/api/auth/sso/start", "", "", 302)
			request(http.MethodGet, appURL+"/api/auth/sso/callback?state=wrong&code=wrong", "", "", 302)
			if getSession().User != nil {
				t.Fatal("failed callback created a session")
			}

			h, _ := request(http.MethodGet, appURL+"/api/auth/sso/start", "", "", 302)
			authorize, err := url.Parse(h.Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if authorize.Query().Get("scope") != "openid email profile" || authorize.Query().Get("code_challenge_method") != "S256" {
				t.Fatalf("authorization=%s", authorize)
			}
			h, _ = request(http.MethodGet, authorize.String(), "", "", 302)
			callback := h.Get("Location")
			h, _ = request(http.MethodGet, callback, "", "", 302)
			if h.Get("Location") != "/" {
				t.Fatalf("callback redirect=%s", h.Get("Location"))
			}
			session := getSession()
			if session.User == nil || session.User.Subject != "staff-123" || session.User.Name != "Jo Example" || session.User.Email != email {
				t.Fatalf("session=%+v", session)
			}
			if !strings.Contains(h.Get("Set-Cookie"), "HttpOnly") || !strings.Contains(h.Get("Set-Cookie"), "SameSite=Lax") {
				t.Fatal("missing session cookie attributes")
			}
			request(http.MethodGet, appURL+"/api/form", "", "", 200)
			_, rendered := request(http.MethodPost, appURL+"/api/render", `{"values":{"display_name":"Reception","contact":"reception@example.org","base":"city"}}`, appURL, 200)
			if !bytes.Contains(rendered, []byte("reception@example.org")) {
				t.Fatalf("shared mailbox signature=%s", rendered)
			}
			if getSession().User.Name != "Jo Example" {
				t.Fatal("signature edits changed authenticated identity")
			}
			request(http.MethodDelete, appURL+"/api/session", "", "https://other.example.org", 403)
			if getSession().User == nil {
				t.Fatal("cross-origin logout removed session")
			}
			request(http.MethodDelete, appURL+"/api/session", "", appURL, 204)
			if getSession().User != nil {
				t.Fatal("logout retained session")
			}
			request(http.MethodGet, appURL+"/api/form", "", "", 401)
			request(http.MethodGet, callback, "", "", 302)
			if getSession().User != nil {
				t.Fatal("replayed callback restored session")
			}
		})
	}
}

func oidcIssuer(t *testing.T, email, callbackURL string) *httptest.Server {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/authorize":
			q := r.URL.Query()
			if q.Get("redirect_uri") != callbackURL {
				t.Error("unexpected redirect URI")
				http.Error(w, "invalid callback", 400)
				return
			}
			target, err := url.Parse(callbackURL)
			if err != nil {
				t.Error(err)
				http.Error(w, "invalid callback", 400)
				return
			}
			target.RawQuery = url.Values{"state": {q.Get("state")}, "code": {q.Get("nonce") + "." + q.Get("code_challenge")}}.Encode()
			http.Redirect(w, r, target.String(), http.StatusFound)
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			nonce, challenge, ok := strings.Cut(r.Form.Get("code"), ".")
			client, secret, authenticated := r.BasicAuth()
			if !ok || challenge != oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")) || !authenticated || client != "sigweft" || secret != "synthetic-secret" {
				t.Error("invalid code exchange")
				http.Error(w, "invalid exchange", 400)
				return
			}
			claims, err := json.Marshal(map[string]any{"iss": issuer, "sub": "staff-123", "aud": "sigweft", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": nonce, "name": "Jo Example", "email": email})
			if err != nil {
				t.Error(err)
				return
			}
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test"}`))
			unsigned := header + "." + base64.RawURLEncoding.EncodeToString(claims)
			digest := sha256.Sum256([]byte(unsigned))
			sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
			if err != nil {
				t.Error(err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "synthetic-token", "token_type": "Bearer", "id_token": unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)})
		default:
			t.Errorf("unexpected identity-provider request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	issuer = server.URL
	t.Cleanup(server.Close)
	return server
}
