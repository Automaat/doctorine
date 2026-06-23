package auth

import (
	"context"
	"errors"
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

func TestAuthenticateReturnsServerErrorOnLookupFailure(t *testing.T) {
	// A transient backend error must not masquerade as an invalid session.
	h := Authenticate(&fakeSessions{lookupErr: errors.New("db unavailable")})(okHandler(nil))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "live-token"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
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

func TestAuthenticateRoutesPersonalToken(t *testing.T) {
	want := &User{ID: 5, Username: "coach"}
	store := &fakeSessions{patUser: want, patScope: ScopeFull, lookupErr: errors.New("session path must not run")}
	var got *User
	h := Authenticate(store)(okHandler(&got))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/results/latest", http.NoBody)
	req.Header.Set("authorization", "Bearer "+PersonalTokenPrefix+"abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if got == nil || got.ID != want.ID {
		t.Fatalf("context user = %v, want %v", got, want)
	}
	if store.patHashLooked != HashSessionToken(PersonalTokenPrefix+"abc") {
		t.Fatal("personal token store did not receive the hashed token")
	}
}

func TestAuthenticateReadScopeBlocksWrites(t *testing.T) {
	store := &fakeSessions{patUser: &User{ID: 5}, patScope: ScopeRead}
	h := Authenticate(store)(okHandler(nil))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/examinations", http.NoBody)
	req.Header.Set("authorization", "Bearer "+PersonalTokenPrefix+"abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAuthenticateReadScopeAllowsReads(t *testing.T) {
	store := &fakeSessions{patUser: &User{ID: 5}, patScope: ScopeRead}
	h := Authenticate(store)(okHandler(nil))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/results/latest", http.NoBody)
	req.Header.Set("authorization", "Bearer "+PersonalTokenPrefix+"abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAuthenticateRejectsRevokedPersonalToken(t *testing.T) {
	store := &fakeSessions{patLookupErr: ErrNotFound}
	h := Authenticate(store)(okHandler(nil))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/results/latest", http.NoBody)
	req.Header.Set("authorization", "Bearer "+PersonalTokenPrefix+"revoked")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
