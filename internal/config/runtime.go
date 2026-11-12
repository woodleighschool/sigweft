package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Runtime configures the HTTP server and identity provider.
type Runtime struct {
	Config              string   `env:"CONFIG" envDefault:"config.yaml"`
	Listen              string   `env:"LISTEN" envDefault:":8080"`
	ServerURL           string   `env:"URL"`
	SessionCookieSecure bool     `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	OIDCIssuerURL       string   `env:"OIDC_ISSUER_URL"`
	OIDCClientID        string   `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret    string   `env:"OIDC_CLIENT_SECRET"`
	OIDCRedirectURL     string   `env:"OIDC_REDIRECT_URL"`
	OIDCScopes          []string `env:"OIDC_SCOPES" envDefault:"openid,email,profile"`
	OIDCEmailClaim      string   `env:"OIDC_EMAIL_CLAIM" envDefault:"email"`
}

// LoadRuntime reads SIGWEFT_* settings and rejects incomplete SSO configuration.
func LoadRuntime() (Runtime, error) {
	var cfg Runtime
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "SIGWEFT_"}); err != nil {
		return cfg, fmt.Errorf("read environment: %w", err)
	}
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	cfg.OIDCIssuerURL = strings.TrimSpace(cfg.OIDCIssuerURL)
	cfg.OIDCClientID = strings.TrimSpace(cfg.OIDCClientID)
	cfg.OIDCRedirectURL = strings.TrimSpace(cfg.OIDCRedirectURL)
	cfg.OIDCEmailClaim = strings.TrimSpace(cfg.OIDCEmailClaim)
	if cfg.OIDCRedirectURL == "" && cfg.ServerURL != "" {
		cfg.OIDCRedirectURL = cfg.ServerURL + "/api/auth/sso/callback"
	}
	if cfg.OIDCIssuerURL == "" && cfg.OIDCClientID == "" && cfg.OIDCClientSecret == "" {
		return cfg, nil
	}
	if cfg.OIDCIssuerURL == "" || cfg.OIDCClientID == "" || cfg.OIDCClientSecret == "" {
		return cfg, fmt.Errorf("SSO requires OIDC_ISSUER_URL, OIDC_CLIENT_ID and OIDC_CLIENT_SECRET")
	}
	issuer, err := url.Parse(cfg.OIDCIssuerURL)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" {
		return cfg, fmt.Errorf("OIDC_ISSUER_URL must be an HTTPS issuer URL")
	}
	redirect, err := url.Parse(cfg.OIDCRedirectURL)
	if err != nil || !validCallbackURL(redirect) {
		return cfg, fmt.Errorf("set URL or OIDC_REDIRECT_URL to an HTTPS address ending in /api/auth/sso/callback (HTTP is allowed on localhost)")
	}
	if len(cfg.OIDCScopes) == 0 || cfg.OIDCEmailClaim == "" {
		return cfg, fmt.Errorf("OIDC_SCOPES and OIDC_EMAIL_CLAIM cannot be empty")
	}
	return cfg, nil
}

// OIDCEnabled reports whether SSO is configured.
func (cfg Runtime) OIDCEnabled() bool {
	return cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" && cfg.OIDCClientSecret != ""
}

func validCallbackURL(u *url.URL) bool {
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path != "/api/auth/sso/callback" {
		return false
	}
	return u.Scheme == "https" || u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}
