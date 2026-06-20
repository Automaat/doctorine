package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler(captured **User) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			user, _ := UserFrom(r.Context())
			*captured = user
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthenticateRejectsMissingToken(t *testing.T) {
	h := Authenticate(&fakeSessions{})(okHandler(nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticateRejectsRevokedOrExpiredSession(t *testing.T) {
	// SessionUser returns ErrNotFound when the row is revoked or expired.
	h := Authenticate(&fakeSessions{lookupErr: ErrNotFound})(okHandler(nil))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "stale-token"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticateAllowsLiveSession(t *testing.T) {
	want := &User{ID: 3, Username: "admin"}
	var got *User
	h := Authenticate(&fakeSessions{user: want})(okHandler(&got))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	req.Header.Set("authorization", "Bearer live-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got == nil || got.Username != want.Username {
		t.Fatalf("context user = %v, want %v", got, want)
	}
}
