package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

// TestOIDCLoginCreatesStoredSession runs the OIDC callback against the real
// session store and checks the resulting cookie authenticates through the
// same middleware as password sessions. It needs a throwaway Postgres and is
// skipped unless DOCTORINE_TEST_DATABASE_URL is set.
func TestOIDCLoginCreatesStoredSession(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the OIDC session DB test")
	}
	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(pool)
	hash, err := HashPassword("oidc-db-test")
	if err != nil {
		t.Fatal(err)
	}
	const username = "oidc-db-test-user"
	if err := store.UpsertAdmin(ctx, username, hash); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	fp := newFakeProvider(t)
	o := NewOIDCHandler(&OIDCConfig{
		IssuerURL:    fp.srv.URL,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirectURL,
		StateSecret:  "state-secret",
	}, NewHandler(store, true, slog.New(slog.DiscardHandler)))

	ar := startOIDCLogin(t, o, fp.srv.URL)
	fp.grant("code-db", fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: username})
	rec := callback(o, url.Values{"code": {"code-db"}, "state": {ar.state}}, ar.cookie)
	if got := rec.Header().Get("Location"); got != "/" {
		t.Fatalf("callback Location = %q, want /", got)
	}
	session := findCookie(rec.Result().Cookies(), CookieName)
	if session == nil {
		t.Fatal("no session cookie set")
	}

	me := meWithSession(store, session)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"`+username+`"`) {
		t.Fatalf("protected endpoint with OIDC session = %d %s", me.Code, me.Body.String())
	}

	if err := store.RevokeSession(ctx, HashSessionToken(session.Value)); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if me := meWithSession(store, session); me.Code != http.StatusUnauthorized {
		t.Fatalf("revoked OIDC session status = %d, want 401", me.Code)
	}
}
