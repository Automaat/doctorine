package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type fakeTokens struct {
	created     PersonalToken
	createErr   error
	gotName     string
	gotScope    string
	gotExpires  *time.Time
	gotHash     string
	list        []PersonalToken
	listErr     error
	revokeErr   error
	revokedID   int
	revokedUser int
}

func (f *fakeTokens) CreatePersonalToken(
	_ context.Context, userID int, tokenHash, name, scope string, expiresAt *time.Time,
) (PersonalToken, error) {
	f.gotName, f.gotScope, f.gotExpires, f.gotHash = name, scope, expiresAt, tokenHash
	if f.createErr != nil {
		return PersonalToken{}, f.createErr
	}
	f.created = PersonalToken{ID: 1, UserID: userID, Name: name, Scope: scope, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}
	return f.created, nil
}

func (f *fakeTokens) ListPersonalTokens(_ context.Context, _ int) ([]PersonalToken, error) {
	return f.list, f.listErr
}

func (f *fakeTokens) RevokePersonalToken(_ context.Context, userID, id int) error {
	f.revokedUser, f.revokedID = userID, id
	return f.revokeErr
}

func (f *fakeTokens) PersonalTokenUser(_ context.Context, _ string) (*User, string, error) {
	return nil, "", ErrNotFound
}

func tokenHandler(tokens PersonalTokenStore) *Handler {
	return &Handler{tokens: tokens, logger: slog.Default()}
}

func withUser(req *http.Request, user *User) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), userKey{}, user))
}

func postToken(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tokens", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req = withUser(req, &User{ID: 9, Username: "admin"})
	rec := httptest.NewRecorder()
	h.CreateToken(rec, req)
	return rec
}

func TestCreateTokenReturnsRawSecretOnce(t *testing.T) {
	tokens := &fakeTokens{}
	rec := postToken(t, tokenHandler(tokens), `{"name":"coach","scope":"read"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	var resp createTokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp.Token, PersonalTokenPrefix) {
		t.Fatalf("token %q missing prefix %q", resp.Token, PersonalTokenPrefix)
	}
	if resp.Scope != ScopeRead {
		t.Fatalf("scope = %q, want read", resp.Scope)
	}
	if tokens.gotHash == "" || tokens.gotHash == resp.Token {
		t.Fatal("store must receive the hash, not the raw token")
	}
	if tokens.gotHash != HashSessionToken(resp.Token) {
		t.Fatal("stored hash does not match the issued token")
	}
}

func TestCreateTokenDefaultsScopeToFull(t *testing.T) {
	tokens := &fakeTokens{}
	rec := postToken(t, tokenHandler(tokens), `{"name":"coach"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if tokens.gotScope != ScopeFull {
		t.Fatalf("scope = %q, want full", tokens.gotScope)
	}
}

func TestCreateTokenRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"missing name": `{"name":"  "}`,
		"bad scope":    `{"name":"x","scope":"admin"}`,
		"past expiry":  `{"name":"x","expires_at":"2000-01-01"}`,
		"bad expiry":   `{"name":"x","expires_at":"not-a-date"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := postToken(t, tokenHandler(&fakeTokens{}), body)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateTokenRequiresUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tokens", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	tokenHandler(&fakeTokens{}).CreateToken(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestListTokensReturnsMetadataOnly(t *testing.T) {
	expires := time.Now().UTC().Add(48 * time.Hour)
	tokens := &fakeTokens{list: []PersonalToken{{ID: 1, Name: "coach", Scope: ScopeRead, ExpiresAt: &expires, CreatedAt: time.Now().UTC()}}}
	req := withUser(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tokens", http.NoBody), &User{ID: 9})
	rec := httptest.NewRecorder()
	tokenHandler(tokens).ListTokens(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "token_hash") || strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("list leaked a secret: %s", rec.Body.String())
	}
	var out []tokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "coach" {
		t.Fatalf("out = %+v", out)
	}
}

func revokeRequest(id string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/tokens/"+id, http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return withUser(req, &User{ID: 9})
}

func TestRevokeToken(t *testing.T) {
	tokens := &fakeTokens{}
	rec := httptest.NewRecorder()
	tokenHandler(tokens).RevokeToken(rec, revokeRequest("7"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (%s)", rec.Code, rec.Body.String())
	}
	if tokens.revokedID != 7 || tokens.revokedUser != 9 {
		t.Fatalf("revoked id=%d user=%d, want 7/9", tokens.revokedID, tokens.revokedUser)
	}
}

func TestRevokeTokenNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	tokenHandler(&fakeTokens{revokeErr: ErrNotFound}).RevokeToken(rec, revokeRequest("7"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestRevokeTokenRejectsBadID(t *testing.T) {
	rec := httptest.NewRecorder()
	tokenHandler(&fakeTokens{}).RevokeToken(rec, revokeRequest("abc"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestParseTokenExpiry(t *testing.T) {
	if got, detail := parseTokenExpiry(nil); got != nil || detail != "" {
		t.Fatalf("nil expiry = (%v, %q), want (nil, \"\")", got, detail)
	}
	empty := "  "
	if got, detail := parseTokenExpiry(&empty); got != nil || detail != "" {
		t.Fatalf("blank expiry = (%v, %q), want (nil, \"\")", got, detail)
	}
	future := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	if got, detail := parseTokenExpiry(&future); got == nil || detail != "" {
		t.Fatalf("future RFC3339 = (%v, %q), want a time", got, detail)
	}
	past := "2000-01-01"
	if _, detail := parseTokenExpiry(&past); detail == "" {
		t.Fatal("past date should be rejected")
	}
}
