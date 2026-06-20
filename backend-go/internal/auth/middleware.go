package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const CookieName = "doctorine_token"

type userKey struct{}

// UserFrom returns the authenticated user stored by Authenticate.
func UserFrom(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userKey{}).(*User)
	return user, ok
}

// Authenticate resolves the presented opaque token to a live server session and
// stores the owning user in the request context. Missing, revoked, or expired
// sessions are rejected with 401.
func Authenticate(sessions SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			user, err := sessions.SessionUser(r.Context(), HashSessionToken(raw))
			if err != nil {
				// Only a missing/revoked/expired session is an auth failure.
				// Treat other (e.g. transient DB) errors as 500 so the
				// frontend keeps the cookie instead of logging the user out.
				if errors.Is(err, ErrNotFound) {
					httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
					return
				}
				slog.Default().Error("load session", "err", err)
				httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			ctx := context.WithValue(r.Context(), userKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tokenFromRequest(r *http.Request) string {
	if raw := bearerToken(r.Header.Get("authorization")); raw != "" {
		return raw
	}
	if cookie, err := r.Cookie(CookieName); err == nil {
		return cookie.Value
	}
	return ""
}

func bearerToken(header string) string {
	prefix, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
