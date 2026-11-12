package config

import "testing"

func TestRuntimeSSOConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		env       map[string]string
		enabled   bool
		wantError bool
	}{
		{name: "public"},
		{name: "partial credentials", env: map[string]string{"OIDC_CLIENT_ID": "client"}, wantError: true},
		{name: "tenant issuer", env: map[string]string{"OIDC_ISSUER_URL": "https://identity.example.org/tenant", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic", "URL": "https://signatures.example.org"}, enabled: true},
		{name: "local callback", env: map[string]string{"OIDC_ISSUER_URL": "https://identity.example.org/tenant", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic", "URL": "http://localhost:5177"}, enabled: true},
		{name: "missing callback", env: map[string]string{"OIDC_ISSUER_URL": "https://identity.example.org", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic"}, wantError: true},
		{name: "insecure issuer", env: map[string]string{"OIDC_ISSUER_URL": "http://identity.example.org", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic", "URL": "https://app.example.org"}, wantError: true},
		{name: "insecure callback", env: map[string]string{"OIDC_ISSUER_URL": "https://identity.example.org", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic", "URL": "http://app.example.org"}, wantError: true},
		{name: "wrong callback path", env: map[string]string{"OIDC_ISSUER_URL": "https://identity.example.org", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "synthetic", "OIDC_REDIRECT_URL": "https://app.example.org/elsewhere"}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL", "URL"} {
				t.Setenv("SIGWEFT_"+key, tc.env[key])
			}
			cfg, err := LoadRuntime()
			if (err != nil) != tc.wantError {
				t.Fatalf("load error=%v", err)
			}
			if err == nil && cfg.OIDCEnabled() != tc.enabled {
				t.Fatalf("enabled=%v", cfg.OIDCEnabled())
			}
			if tc.enabled && cfg.OIDCRedirectURL != tc.env["URL"]+"/api/auth/sso/callback" {
				t.Fatalf("callback=%s", cfg.OIDCRedirectURL)
			}
		})
	}
}

func TestPrefillFieldValidation(t *testing.T) {
	for _, tc := range []struct {
		name, kind, prefill string
		readOnly, wantError bool
	}{
		{"name", "text", "name", false, false},
		{"email", "email", "email", false, false},
		{"unknown claim", "text", "subject", false, true},
		{"list", "text-list", "name", false, true},
		{"fixed field", "text", "name", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := Field{Label: "Identity", Type: tc.kind, Prefill: tc.prefill, ReadOnly: tc.readOnly, Default: ""}
			if err := field.validate(); (err != nil) != tc.wantError {
				t.Fatalf("validate error=%v", err)
			}
		})
	}
}
