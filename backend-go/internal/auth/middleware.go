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

// Authenticator resolves presented tokens to their owning user. Session tokens
// and long-lived personal access tokens are looked up through separate stores.
type Authenticator interface {
	SessionUser(ctx context.Context, tokenHash string) (*User, error)
	PersonalTokenUser(ctx context.Context, tokenHash string) (*User, string, error)
}

// UserFrom returns the authenticated user stored by Authenticate.
func UserFrom(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userKey{}).(*User)
	return user, ok
}

// Authenticate resolves the presented token to its owning user and stores the
// user in the request context. A token carrying PersonalTokenPrefix is resolved
// as a long-lived personal access token (read-scoped tokens are limited to safe
// methods); anything else is treated as a session token. Missing, revoked, or
// expired credentials are rejected with 401.
func Authenticate(store Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}

			var user *User
			var err error
			if strings.HasPrefix(raw, PersonalTokenPrefix) {
				var scope string
				user, scope, err = store.PersonalTokenUser(r.Context(), HashSessionToken(raw))
				if err == nil && scope == ScopeRead && !isReadMethod(r.Method) {
					httputil.WriteDetailError(w, http.StatusForbidden, "Token is read-only")
					return
				}
			} else {
				user, err = store.SessionUser(r.Context(), HashSessionToken(raw))
			}
			if err != nil {
				// Only a missing/revoked/expired credential is an auth failure.
				// Treat other (e.g. transient DB) errors as 500 so the
				// frontend keeps the cookie instead of logging the user out.
				if errors.Is(err, ErrNotFound) {
					httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
					return
				}
				slog.Default().Error("authenticate request", "err", err)
				httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			ctx := context.WithValue(r.Context(), userKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isReadMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
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
