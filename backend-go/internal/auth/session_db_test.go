package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

// TestSessionStoreRejectsRevokedAndExpired exercises the real SQL predicate that
// keeps revoked and expired sessions out of SessionUser. It needs a throwaway
// Postgres and is skipped unless DOCTORINE_TEST_DATABASE_URL is set, so it never
// runs in the unit-test CI job (which has no database).
func TestSessionStoreRejectsRevokedAndExpired(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the session store DB test")
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

	hash, err := HashPassword("session-store-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertAdmin(ctx, "session-store-test-user", hash); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	user, err := store.GetByUsername(ctx, "session-store-test-user")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}

	_, liveHash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate live token: %v", err)
	}
	if err := store.CreateSession(ctx, user.ID, liveHash, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("create live session: %v", err)
	}
	if got, err := store.SessionUser(ctx, liveHash); err != nil || got.ID != user.ID {
		t.Fatalf("live session lookup = (%v, %v), want user %d", got, err, user.ID)
	}

	if err := store.RevokeSession(ctx, liveHash); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := store.SessionUser(ctx, liveHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked session lookup err = %v, want ErrNotFound", err)
	}

	_, expiredHash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	if err := store.CreateSession(ctx, user.ID, expiredHash, time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	if _, err := store.SessionUser(ctx, expiredHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session lookup err = %v, want ErrNotFound", err)
	}
}
