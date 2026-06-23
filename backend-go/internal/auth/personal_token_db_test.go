package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

// TestPersonalTokenStore exercises the real SQL behind personal access tokens:
// creation, scoped lookup with last_used_at touch, owner-scoped revocation, and
// rejection of revoked/expired tokens. It needs a throwaway Postgres and is
// skipped unless DOCTORINE_TEST_DATABASE_URL is set.
func TestPersonalTokenStore(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the personal token store DB test")
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

	hash, err := HashPassword("pat-store-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertAdmin(ctx, "pat-store-test-user", hash); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	user, err := store.GetByUsername(ctx, "pat-store-test-user")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}

	_, liveHash, err := GeneratePersonalToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	token, err := store.CreatePersonalToken(ctx, user.ID, liveHash, "coach", ScopeRead, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if token.Scope != ScopeRead || token.Name != "coach" || token.LastUsedAt != nil {
		t.Fatalf("unexpected created token: %+v", token)
	}

	got, scope, err := store.PersonalTokenUser(ctx, liveHash)
	if err != nil || got.ID != user.ID || scope != ScopeRead {
		t.Fatalf("token lookup = (%v, %q, %v), want user %d read", got, scope, err, user.ID)
	}

	listed, err := store.ListPersonalTokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *PersonalToken
	for i := range listed {
		if listed[i].ID == token.ID {
			found = &listed[i]
		}
	}
	if found == nil {
		t.Fatalf("created token %d not returned by list", token.ID)
	}
	if found.LastUsedAt == nil {
		t.Fatal("last_used_at was not recorded on use")
	}

	// A different user cannot revoke this token.
	if err := store.RevokePersonalToken(ctx, user.ID+9999, token.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user revoke err = %v, want ErrNotFound", err)
	}
	if err := store.RevokePersonalToken(ctx, user.ID, token.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, _, err := store.PersonalTokenUser(ctx, liveHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked token lookup err = %v, want ErrNotFound", err)
	}

	// Expired tokens are rejected too.
	_, expHash, err := GeneratePersonalToken()
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	past := time.Now().UTC().Add(-time.Hour)
	if _, err := store.CreatePersonalToken(ctx, user.ID, expHash, "old", ScopeFull, &past); err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	if _, _, err := store.PersonalTokenUser(ctx, expHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired token lookup err = %v, want ErrNotFound", err)
	}
}
