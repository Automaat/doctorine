package auth

import (
	"maps"
	"strings"
	"testing"
)

func TestLoadOIDCConfig(t *testing.T) {
	full := map[string]string{
		"DOCTORINE_OIDC_ISSUER_URL":    "https://auth.example.test",
		"DOCTORINE_OIDC_CLIENT_ID":     "doctorine",
		"DOCTORINE_OIDC_CLIENT_SECRET": "client-secret",
		"DOCTORINE_OIDC_REDIRECT_URL":  "https://doctorine.example.test/api/auth/oidc/callback",
	}
	with := func(overrides map[string]string) map[string]string {
		env := maps.Clone(full)
		maps.Copy(env, overrides)
		return env
	}

	tests := []struct {
		name            string
		env             map[string]string
		wantEnabled     bool
		wantErrContains []string
		wantStateSecret string
	}{
		{
			name: "none set disables OIDC",
			env:  map[string]string{},
		},
		{
			name: "whitespace-only values count as unset",
			env:  map[string]string{"DOCTORINE_OIDC_ISSUER_URL": "  "},
		},
		{
			name:            "all set enables OIDC and keys state with JWT secret",
			env:             with(map[string]string{"DOCTORINE_JWT_SECRET": "jwt-secret"}),
			wantEnabled:     true,
			wantStateSecret: "jwt-secret",
		},
		{
			name:            "state key falls back to client secret without JWT secret",
			env:             with(nil),
			wantEnabled:     true,
			wantStateSecret: "client-secret",
		},
		{
			name:            "missing client secret is an error",
			env:             with(map[string]string{"DOCTORINE_OIDC_CLIENT_SECRET": ""}),
			wantErrContains: []string{"DOCTORINE_OIDC_CLIENT_SECRET missing"},
		},
		{
			name: "only issuer set names the three missing variables",
			env:  map[string]string{"DOCTORINE_OIDC_ISSUER_URL": "https://auth.example.test"},
			wantErrContains: []string{
				"DOCTORINE_OIDC_ISSUER_URL set",
				"DOCTORINE_OIDC_CLIENT_ID", "DOCTORINE_OIDC_CLIENT_SECRET", "DOCTORINE_OIDC_REDIRECT_URL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, enabled, err := LoadOIDCConfig(func(key string) string { return tt.env[key] })

			if len(tt.wantErrContains) > 0 {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tt.wantErrContains)
				}
				for _, want := range tt.wantErrContains {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("err = %q, want it to contain %q", err, want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if enabled != tt.wantEnabled {
				t.Fatalf("enabled = %v, want %v", enabled, tt.wantEnabled)
			}
			if !enabled {
				return
			}
			want := OIDCConfig{
				IssuerURL:    full["DOCTORINE_OIDC_ISSUER_URL"],
				ClientID:     full["DOCTORINE_OIDC_CLIENT_ID"],
				ClientSecret: full["DOCTORINE_OIDC_CLIENT_SECRET"],
				RedirectURL:  full["DOCTORINE_OIDC_REDIRECT_URL"],
				StateSecret:  tt.wantStateSecret,
			}
			if cfg != want {
				t.Fatalf("cfg = %+v, want %+v", cfg, want)
			}
		})
	}
}
