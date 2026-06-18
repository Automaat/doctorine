package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const CookieName = "doctorine_token"

type claimsKey struct{}

func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*Claims)
	return claims, ok
}

func Authenticate(tokens *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r.Header.Get("authorization"))
			if raw == "" {
				if cookie, err := r.Cookie(CookieName); err == nil {
					raw = cookie.Value
				}
			}
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			claims, err := tokens.Parse(raw)
			if err != nil {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) string {
	prefix, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
