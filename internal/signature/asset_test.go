package signature

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	htmltext "html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func assetConfig(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	tmpl := `<img src="{{.Assets.logo.Src}}" alt="{{.Assets.logo.Alt}}">`
	if err := os.WriteFile(filepath.Join(dir, "signature.html"), []byte(tmpl), 0600); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`title: Example
template: signature.html
fields: [{id: name, label: Name, type: text}]
assets:
  logo:
    src: %q
    alt: Example
    width: 48
    height: 48
`, src)
	path := filepath.Join(dir, "sigweft.yaml")
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRemoteAssetLoadedOnceAndEmbedded(t *testing.T) {
	data, err := os.ReadFile("../../examples/studio/assets/emblem.png")
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logo" {
			http.Redirect(w, r, "/logo.png", http.StatusFound)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	renderer, err := Load(t.Context(), assetConfig(t, server.URL+"/logo"))
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	want := `src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(data) + `"`
	for range 2 {
		result, err := renderer.Render(nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(htmltext.UnescapeString(result.HTML), want) || strings.Contains(result.HTML, server.URL) {
			t.Fatalf("rendered image does not contain the original bytes: %.200q", result.HTML)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("image requests = %d, want 1", got)
	}
}

func TestRemoteAssetFailuresPreventLoading(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name: "HTTP failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			want: "HTTP status 503",
		},
		{
			name: "invalid image",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write([]byte("not an image"))
			},
			want: "decode image",
		},
		{
			name: "declared size",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(maxAssetBytes+1))
			},
			want: "image exceeds 2 MiB",
		},
		{
			name: "streamed size",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.(http.Flusher).Flush()
				_, _ = w.Write(bytes.Repeat([]byte("x"), maxAssetBytes+1))
			},
			want: "image exceeds 2 MiB",
		},
		{
			name: "redirect loop",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/", http.StatusFound)
			},
			want: "too many redirects",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			t.Cleanup(server.Close)
			_, err := Load(t.Context(), assetConfig(t, server.URL))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRemoteAssetLoadCanBeCancelled(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	path := assetConfig(t, server.URL)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Load(ctx, path)
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Load error = %v, want context cancellation", err)
	}
}

func TestAssetSourceRequiresLocalPathOrHTTPURL(t *testing.T) {
	for _, src := range []string{"file:///tmp/logo.png", "ftp://example.org/logo.png", "https:/logo.png", "//example.org/logo.png"} {
		t.Run(src, func(t *testing.T) {
			_, err := Load(t.Context(), assetConfig(t, src))
			if err == nil || !strings.Contains(err.Error(), "src must be") {
				t.Fatalf("Load error = %v, want invalid source", err)
			}
		})
	}
}
