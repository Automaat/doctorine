package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const (
	testClientID     = "doctorine"
	testClientSecret = "test-client-secret"
	testRedirectURL  = "https://doctorine.example.test/api/auth/oidc/callback"
)

// fakeGrant is what the fake provider issues for one authorization code.
type fakeGrant struct {
	challenge       string
	nonce           string
	username        string
	audience        string
	userinfoOnly    bool
	userinfoSubject string
}

// fakeProvider is an in-process OIDC provider: discovery, JWKS and a token
// endpoint that enforces client_secret_basic and PKCE S256 and signs ID tokens.
type fakeProvider struct {
	t      *testing.T
	srv    *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	down   bool
	grants map[string]fakeGrant
	tokens map[string]fakeGrant
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{t: t, key: key, grants: map[string]fakeGrant{}, tokens: map[string]fakeGrant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", fp.discovery)
	mux.HandleFunc("GET /jwks", fp.jwks)
	mux.HandleFunc("POST /token", fp.token)
	mux.HandleFunc("GET /userinfo", fp.userinfo)
	fp.srv = httptest.NewServer(mux)
	t.Cleanup(fp.srv.Close)
	return fp
}

func (fp *fakeProvider) setDown(down bool) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.down = down
}

func (fp *fakeProvider) grant(code string, g fakeGrant) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.grants[code] = g
}

func (fp *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	fp.mu.Lock()
	down := fp.down
	fp.mu.Unlock()
	if down {
		http.Error(w, "down", http.StatusServiceUnavailable)
		return
	}
	base := fp.srv.URL
	writeTestJSON(w, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/authorize",
		"token_endpoint":                        base + "/token",
		"jwks_uri":                              base + "/jwks",
		"userinfo_endpoint":                     base + "/userinfo",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (fp *fakeProvider) jwks(w http.ResponseWriter, _ *http.Request) {
	writeTestJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &fp.key.PublicKey, KeyID: "test", Algorithm: string(jose.RS256), Use: "sig",
	}}})
}

func (fp *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	id, secret, ok := r.BasicAuth()
	if !ok || id != testClientID || secret != testClientSecret {
		writeTokenError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeTokenError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("redirect_uri") != testRedirectURL {
		writeTokenError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	fp.mu.Lock()
	g, found := fp.grants[r.PostForm.Get("code")]
	delete(fp.grants, r.PostForm.Get("code"))
	fp.mu.Unlock()
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if !found || base64.RawURLEncoding.EncodeToString(sum[:]) != g.challenge {
		writeTokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	audience := g.audience
	if audience == "" {
		audience = testClientID
	}
	now := time.Now()
	claims := map[string]any{
		"iss":   fp.srv.URL,
		"sub":   "subject-" + g.username,
		"aud":   audience,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"nonce": g.nonce,
	}
	if !g.userinfoOnly {
		claims["preferred_username"] = g.username
	}
	accessToken := "access-" + r.PostForm.Get("code")
	fp.mu.Lock()
	fp.tokens[accessToken] = g
	fp.mu.Unlock()
	writeTestJSON(w, map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     fp.sign(claims),
	})
}

func (fp *fakeProvider) userinfo(w http.ResponseWriter, r *http.Request) {
	fp.mu.Lock()
	g, ok := fp.tokens[strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")]
	fp.mu.Unlock()
	if !ok {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	subject := g.userinfoSubject
	if subject == "" {
		subject = "subject-" + g.username
	}
	writeTestJSON(w, map[string]any{"sub": subject, "preferred_username": g.username})
}

func (fp *fakeProvider) sign(claims map[string]any) string {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: fp.key, KeyID: "test"}},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		fp.t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		fp.t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		fp.t.Fatal(err)
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		fp.t.Fatal(err)
	}
	return raw
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeTokenError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

// memStore is an in-memory user and session store for handler tests.
type memStore struct {
	mu       sync.Mutex
	users    map[string]*User
	sessions map[string]int
}

func newMemStore(users ...*User) *memStore {
	s := &memStore{users: map[string]*User{}, sessions: map[string]int{}}
	for _, u := range users {
		s.users[u.Username] = u
	}
	return s
}

func (s *memStore) GetByUsername(_ context.Context, username string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[strings.TrimSpace(username)]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (s *memStore) CreateSession(_ context.Context, userID int, tokenHash string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[tokenHash] = userID
	return nil
}

func (s *memStore) SessionUser(_ context.Context, tokenHash string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.sessions[tokenHash]
	if !ok {
		return nil, ErrNotFound
	}
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (s *memStore) RevokeSession(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tokenHash)
	return nil
}

func (s *memStore) PersonalTokenUser(context.Context, string) (*User, string, error) {
	return nil, "", ErrNotFound
}

func (s *memStore) sessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func newTestOIDCHandler(issuer string, users UserStore, sessions SessionStore) *OIDCHandler {
	h := &Handler{
		users:        users,
		sessions:     sessions,
		logger:       slog.New(slog.DiscardHandler),
		cookieSecure: true,
	}
	return NewOIDCHandler(&OIDCConfig{
		IssuerURL:    issuer,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirectURL,
		StateSecret:  "state-secret",
	}, h)
}

// authRequest is what the login endpoint sent the browser to the provider with.
type authRequest struct {
	state     string
	nonce     string
	challenge string
	cookie    *http.Cookie
}

func startOIDCLogin(t *testing.T, o *OIDCHandler, issuer string) authRequest {
	t.Helper()
	rec := httptest.NewRecorder()
	o.Login(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
	if rec.Code != http.StatusFound {
		t.Fatalf("login status = %d, want 302", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if got := loc.Scheme + "://" + loc.Host + loc.Path; got != issuer+"/authorize" {
		t.Fatalf("login redirect = %q, want provider authorize endpoint", got)
	}
	q := loc.Query()
	wantParams := map[string]string{
		"response_type":         "code",
		"client_id":             testClientID,
		"redirect_uri":          testRedirectURL,
		"scope":                 "openid profile email",
		"code_challenge_method": "S256",
	}
	for k, want := range wantParams {
		if got := q.Get(k); got != want {
			t.Errorf("authorize %s = %q, want %q", k, got, want)
		}
	}
	cookie := findCookie(rec.Result().Cookies(), OIDCStateCookieName)
	if cookie == nil {
		t.Fatal("login did not set the state cookie")
	}
	if cookie.Path != "/api/auth/oidc" || !cookie.HttpOnly || !cookie.Secure ||
		cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 600 {
		t.Errorf("state cookie attributes = %+v", cookie)
	}
	return authRequest{state: q.Get("state"), nonce: q.Get("nonce"), challenge: q.Get("code_challenge"), cookie: cookie}
}

func callback(o *OIDCHandler, query url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/auth/oidc/callback?"+query.Encode(), http.NoBody,
	)
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	rec := httptest.NewRecorder()
	o.Callback(rec, req)
	return rec
}

// meWithSession calls a protected endpoint with the given session cookie.
func meWithSession(store Authenticator, session *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: session.Name, Value: session.Value})
	rec := httptest.NewRecorder()
	h := &Handler{}
	Authenticate(store)(http.HandlerFunc(h.Me)).ServeHTTP(rec, req)
	return rec
}

func TestOIDCCallback(t *testing.T) {
	user := &User{ID: 3, Username: "mskalski", CreatedAt: time.Now().UTC()}

	tests := []struct {
		name        string
		grant       func(ar authRequest) fakeGrant
		query       func(ar authRequest) url.Values
		cookie      func(ar authRequest) *http.Cookie
		wantError   string
		wantSession bool
	}{
		{
			name:        "existing user gets a session",
			wantSession: true,
		},
		{
			name: "username from userinfo when the id token omits it",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: user.Username, userinfoOnly: true}
			},
			wantSession: true,
		},
		{
			name: "userinfo for a different subject is rejected",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{
					challenge: ar.challenge, nonce: ar.nonce, username: user.Username,
					userinfoOnly: true, userinfoSubject: "someone-else",
				}
			},
			wantError: OIDCErrorFailed,
		},
		{
			name: "unknown user is rejected without a session",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: "stranger"}
			},
			wantError: OIDCErrorNoUser,
		},
		{
			name: "username differing only by whitespace is not an exact match",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: " mskalski"}
			},
			wantError: OIDCErrorNoUser,
		},
		{
			name: "bad state",
			query: func(ar authRequest) url.Values {
				return url.Values{"code": {"code-1"}, "state": {ar.state + "x"}}
			},
			wantError: OIDCErrorFailed,
		},
		{
			name: "nonce mismatch",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: ar.challenge, nonce: "other-nonce", username: user.Username}
			},
			wantError: OIDCErrorFailed,
		},
		{
			name: "wrong audience",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: user.Username, audience: "other-app"}
			},
			wantError: OIDCErrorFailed,
		},
		{
			name: "PKCE verifier does not match challenge",
			grant: func(ar authRequest) fakeGrant {
				return fakeGrant{challenge: "not-the-challenge", nonce: ar.nonce, username: user.Username}
			},
			wantError: OIDCErrorFailed,
		},
		{
			name:      "missing state cookie",
			cookie:    func(authRequest) *http.Cookie { return nil },
			wantError: OIDCErrorFailed,
		},
		{
			name: "tampered state cookie",
			cookie: func(ar authRequest) *http.Cookie {
				c := *ar.cookie
				c.Value = "x" + c.Value
				return &c
			},
			wantError: OIDCErrorFailed,
		},
		{
			name: "provider returned an error",
			query: func(ar authRequest) url.Values {
				return url.Values{"error": {"access_denied"}, "state": {ar.state}}
			},
			wantError: OIDCErrorFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := newFakeProvider(t)
			store := newMemStore(user)
			o := newTestOIDCHandler(fp.srv.URL, store, store)
			ar := startOIDCLogin(t, o, fp.srv.URL)

			g := fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: user.Username}
			if tt.grant != nil {
				g = tt.grant(ar)
			}
			fp.grant("code-1", g)
			query := url.Values{"code": {"code-1"}, "state": {ar.state}}
			if tt.query != nil {
				query = tt.query(ar)
			}
			cookie := ar.cookie
			if tt.cookie != nil {
				cookie = tt.cookie(ar)
			}

			rec := callback(o, query, cookie)

			if rec.Code != http.StatusFound {
				t.Fatalf("status = %d, want 302", rec.Code)
			}
			wantLocation := "/"
			if tt.wantError != "" {
				wantLocation = "/login?error=" + tt.wantError
			}
			if got := rec.Header().Get("Location"); got != wantLocation {
				t.Fatalf("Location = %q, want %q", got, wantLocation)
			}
			cleared := findCookie(rec.Result().Cookies(), OIDCStateCookieName)
			if cleared == nil || cleared.MaxAge >= 0 || cleared.Path != "/api/auth/oidc" {
				t.Errorf("state cookie not cleared: %+v", cleared)
			}

			session := findCookie(rec.Result().Cookies(), CookieName)
			if !tt.wantSession {
				if session != nil || store.sessionCount() != 0 {
					t.Fatalf("unexpected session: cookie %+v, %d sessions", session, store.sessionCount())
				}
				return
			}
			if session == nil {
				t.Fatal("no session cookie set")
			}
			if session.Path != "/" || !session.HttpOnly || !session.Secure || session.SameSite != http.SameSiteLaxMode {
				t.Errorf("session cookie attributes = %+v, want same as password login", session)
			}
			me := meWithSession(store, session)
			if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"mskalski"`) {
				t.Fatalf("protected endpoint with OIDC session = %d %s", me.Code, me.Body.String())
			}
		})
	}
}

func TestOIDCCodeCannotBeReplayed(t *testing.T) {
	user := &User{ID: 3, Username: "mskalski"}
	fp := newFakeProvider(t)
	store := newMemStore(user)
	o := newTestOIDCHandler(fp.srv.URL, store, store)
	ar := startOIDCLogin(t, o, fp.srv.URL)
	fp.grant("code-1", fakeGrant{challenge: ar.challenge, nonce: ar.nonce, username: user.Username})

	first := callback(o, url.Values{"code": {"code-1"}, "state": {ar.state}}, ar.cookie)
	if got := first.Header().Get("Location"); got != "/" {
		t.Fatalf("first callback Location = %q, want /", got)
	}
	replay := callback(o, url.Values{"code": {"code-1"}, "state": {ar.state}}, ar.cookie)
	if got := replay.Header().Get("Location"); got != "/login?error=oidc_failed" {
		t.Fatalf("replayed callback Location = %q, want oidc_failed", got)
	}
}

func TestOIDCProviderDown(t *testing.T) {
	user := &User{ID: 3, Username: "mskalski"}

	t.Run("login while discovery fails, then recovers", func(t *testing.T) {
		fp := newFakeProvider(t)
		fp.setDown(true)
		store := newMemStore(user)
		o := newTestOIDCHandler(fp.srv.URL, store, store)

		rec := httptest.NewRecorder()
		o.Login(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
		if got := rec.Header().Get("Location"); got != "/login?error=oidc_unavailable" {
			t.Fatalf("Location = %q, want oidc_unavailable", got)
		}
		if findCookie(rec.Result().Cookies(), OIDCStateCookieName) != nil {
			t.Error("state cookie set although the provider is down")
		}

		fp.setDown(false)
		startOIDCLogin(t, o, fp.srv.URL)

		fp.setDown(true)
		startOIDCLogin(t, o, fp.srv.URL)
	})

	t.Run("callback while discovery fails", func(t *testing.T) {
		fp := newFakeProvider(t)
		store := newMemStore(user)
		ar := startOIDCLogin(t, newTestOIDCHandler(fp.srv.URL, store, store), fp.srv.URL)
		fp.setDown(true)
		fresh := newTestOIDCHandler(fp.srv.URL, store, store)

		rec := callback(fresh, url.Values{"code": {"code-1"}, "state": {ar.state}}, ar.cookie)
		if got := rec.Header().Get("Location"); got != "/login?error=oidc_unavailable" {
			t.Fatalf("Location = %q, want oidc_unavailable", got)
		}
	})

	t.Run("token endpoint unreachable", func(t *testing.T) {
		fp := newFakeProvider(t)
		store := newMemStore(user)
		o := newTestOIDCHandler(fp.srv.URL, store, store)
		ar := startOIDCLogin(t, o, fp.srv.URL)
		fp.srv.Close()

		rec := callback(o, url.Values{"code": {"code-1"}, "state": {ar.state}}, ar.cookie)
		if got := rec.Header().Get("Location"); got != "/login?error=oidc_unavailable" {
			t.Fatalf("Location = %q, want oidc_unavailable", got)
		}
		if store.sessionCount() != 0 {
			t.Fatal("session created although the provider is down")
		}
	})
}

func TestOIDCDisabled(t *testing.T) {
	store := newMemStore()
	h := &Handler{users: store, sessions: store, logger: slog.New(slog.DiscardHandler)}
	o := NewOIDCHandler(nil, h)

	rec := httptest.NewRecorder()
	o.Status(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/oidc/status", http.NoBody))
	if strings.TrimSpace(rec.Body.String()) != `{"enabled":false}` {
		t.Fatalf("status body = %q", rec.Body.String())
	}
	for _, handler := range []http.HandlerFunc{o.Login, o.Callback} {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("disabled flow status = %d, want 404", rec.Code)
		}
	}

	enabled := newTestOIDCHandler("https://auth.example.test", store, store)
	rec = httptest.NewRecorder()
	enabled.Status(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/oidc/status", http.NoBody))
	if strings.TrimSpace(rec.Body.String()) != `{"enabled":true}` {
		t.Fatalf("status body = %q", rec.Body.String())
	}
}
