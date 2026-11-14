package auth

import (
	"fmt"
	"strings"
)

// OIDCConfig holds the settings for OIDC single sign-on. StateSecret keys the
// HMAC over the short-lived state cookie: DOCTORINE_JWT_SECRET when set,
// otherwise the client secret.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	StateSecret  string
}

// LoadOIDCConfig reads the DOCTORINE_OIDC_* settings through getenv. All four
// set enables OIDC; none set reports it disabled; any other combination
// returns an error naming the missing variables.
func LoadOIDCConfig(getenv func(string) string) (OIDCConfig, bool, error) {
	read := func(key string) string { return strings.TrimSpace(getenv(key)) }
	cfg := OIDCConfig{
		IssuerURL:    read("DOCTORINE_OIDC_ISSUER_URL"),
		ClientID:     read("DOCTORINE_OIDC_CLIENT_ID"),
		ClientSecret: read("DOCTORINE_OIDC_CLIENT_SECRET"),
		RedirectURL:  read("DOCTORINE_OIDC_REDIRECT_URL"),
		StateSecret:  getenv("DOCTORINE_JWT_SECRET"),
	}
	var set, missing []string
	for _, field := range []struct{ key, value string }{
		{"DOCTORINE_OIDC_ISSUER_URL", cfg.IssuerURL},
		{"DOCTORINE_OIDC_CLIENT_ID", cfg.ClientID},
		{"DOCTORINE_OIDC_CLIENT_SECRET", cfg.ClientSecret},
		{"DOCTORINE_OIDC_REDIRECT_URL", cfg.RedirectURL},
	} {
		if field.value == "" {
			missing = append(missing, field.key)
		} else {
			set = append(set, field.key)
		}
	}
	if len(set) == 0 {
		return OIDCConfig{}, false, nil
	}
	if len(missing) > 0 {
		return OIDCConfig{}, false, fmt.Errorf(
			"incomplete OIDC config: %s set but %s missing; set all four or none",
			strings.Join(set, ", "), strings.Join(missing, ", "),
		)
	}
	if cfg.StateSecret == "" {
		cfg.StateSecret = cfg.ClientSecret
	}
	return cfg, true, nil
}
