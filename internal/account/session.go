// Package account owns browser sign-in and the current session.
package account

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	authoidc "github.com/woodleighschool/goodies/auth/oidc"
	"github.com/woodleighschool/sigweft/internal/config"
)

// Service stores verified OIDC identities in browser sessions.
type Service struct {
	sessions *scs.SessionManager
	oidc     *authoidc.Client
	logger   *slog.Logger
}

// New configures sessions and completes OIDC discovery when SSO is enabled.
func New(ctx context.Context, cfg config.Runtime, logger *slog.Logger) (*Service, error) {
	sessions := scs.New()
	sessions.Lifetime = 14 * 24 * time.Hour
	sessions.HashTokenInStore = true
	sessions.Cookie.Name = "sigweft_session"
	sessions.Cookie.Path = "/"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = cfg.SessionCookieSecure
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true
	sessions.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.ErrorContext(r.Context(), "session persistence failed", "err", err)
		http.Error(w, "request failed", http.StatusInternalServerError)
	}
	s := &Service{sessions: sessions, logger: logger}
	if cfg.OIDCEnabled() {
		provider, err := authoidc.New(ctx, sessions, authoidc.Config{
			IssuerURL: cfg.OIDCIssuerURL, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret,
			RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes, EmailClaim: cfg.OIDCEmailClaim,
		})
		if err != nil {
			return nil, fmt.Errorf("configure authentication: %w", err)
		}
		s.oidc = provider
	}
	return s, nil
}

// Enabled reports whether requests require a signed-in user.
func (s *Service) Enabled() bool { return s != nil && s.oidc != nil }

// LoadAndSave loads sessions and rejects cross-origin browser mutations.
func (s *Service) LoadAndSave(next http.Handler) http.Handler {
	return s.sessions.LoadAndSave(http.NewCrossOriginProtection().Handler(next))
}

// User returns the signed-in identity, independently of editable signature fields.
func (s *Service) User(ctx context.Context) *authoidc.Identity {
	if !s.Enabled() {
		return nil
	}
	subject := s.sessions.GetString(ctx, "subject")
	if subject == "" {
		return nil
	}
	return &authoidc.Identity{
		Issuer: s.sessions.GetString(ctx, "issuer"), Subject: subject,
		Name: s.sessions.GetString(ctx, "name"), Email: s.sessions.GetString(ctx, "email"),
	}
}

func (s *Service) start(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		http.NotFound(w, r)
		return
	}
	target, err := s.oidc.Begin(r.Context())
	if err != nil {
		s.failed(w, r, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		http.NotFound(w, r)
		return
	}
	query := r.URL.Query()
	identity, err := s.oidc.Complete(r.Context(), query.Get("state"), query.Get("code"))
	if err != nil {
		s.failed(w, r, err)
		return
	}
	if err := s.sessions.RenewToken(r.Context()); err != nil {
		s.failed(w, r, err)
		return
	}
	s.sessions.Put(r.Context(), "issuer", identity.Issuer)
	s.sessions.Put(r.Context(), "subject", identity.Subject)
	s.sessions.Put(r.Context(), "name", identity.Name)
	s.sessions.Put(r.Context(), "email", identity.Email)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Service) failed(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.ErrorContext(r.Context(), "sso sign-in failed", "err", err)
	http.Redirect(w, r, "/login?sso_error=sign_in_failed", http.StatusFound)
}
