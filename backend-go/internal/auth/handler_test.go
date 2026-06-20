package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeSessions records session operations so tests can assert on them without a
// database.
type fakeSessions struct {
	user        *User
	lookupErr   error
	createErr   error
	createdHash string
	createdUser int
	revoked     []string
}

func (f *fakeSessions) CreateSession(_ context.Context, userID int, tokenHash string, _ time.Time) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createdUser = userID
	f.createdHash = tokenHash
	return nil
}

func (f *fakeSessions) SessionUser(_ context.Context, _ string) (*User, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	return f.user, nil
}

func (f *fakeSessions) RevokeSession(_ context.Context, tokenHash string) error {
	f.revoked = append(f.revoked, tokenHash)
	return nil
}

type fakeUsers struct {
	user *User
}

func (f *fakeUsers) GetByUsername(_ context.Context, _ string) (*User, error) {
	if f.user == nil {
		return nil, ErrNotFound
	}
	return f.user, nil
}

func newHandler(users UserStore, sessions SessionStore) *Handler {
	return &Handler{users: users, sessions: sessions, logger: slog.Default()}
}

func TestLoginCreatesRevocableSession(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	user := &User{ID: 7, Username: "admin", PasswordHash: hash, CreatedAt: time.Now().UTC()}
	sessions := &fakeSessions{}
	h := newHandler(&fakeUsers{user: user}, sessions)

	body := strings.NewReader(`{"username":"admin","password":"secret"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/auth/login", body)
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	cookie := findCookie(rec.Result().Cookies(), CookieName)
	if cookie == nil || cookie.Value == "" {
		t.Fatal("login did not set a session cookie")
	}
	if sessions.createdUser != user.ID {
		t.Fatalf("session created for user %d, want %d", sessions.createdUser, user.ID)
	}
	if sessions.createdHash != HashSessionToken(cookie.Value) {
		t.Fatal("stored session hash does not match the cookie token")
	}
}

func TestLoginRejectsBadPassword(t *testing.T) {
	hash, _ := HashPassword("secret")
	user := &User{ID: 1, Username: "admin", PasswordHash: hash, CreatedAt: time.Now().UTC()}
	h := newHandler(&fakeUsers{user: user}, &fakeSessions{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestLogoutRevokesPresentedToken(t *testing.T) {
	sessions := &fakeSessions{}
	h := newHandler(&fakeUsers{}, sessions)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/auth/logout", http.NoBody)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "opaque-token"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if len(sessions.revoked) != 1 || sessions.revoked[0] != HashSessionToken("opaque-token") {
		t.Fatalf("revoked = %v, want one entry for the presented token hash", sessions.revoked)
	}
	cleared := findCookie(rec.Result().Cookies(), CookieName)
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Fatal("logout did not clear the session cookie")
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
