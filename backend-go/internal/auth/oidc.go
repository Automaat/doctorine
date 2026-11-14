package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const (
	OIDCStateCookieName = "doctorine_oidc_state"
	oidcCookiePath      = "/api/auth/oidc"
	oidcStateTTL        = 10 * time.Minute
	oidcProviderTimeout = 10 * time.Second

	OIDCErrorFailed      = "oidc_failed"
	OIDCErrorNoUser      = "oidc_no_user"
	OIDCErrorUnavailable = "oidc_unavailable"
)

var errInvalidStateCookie = errors.New("invalid OIDC state cookie")

// OIDCHandler serves the OIDC authorization-code login flow. Provider
// discovery is lazy and retried on every request until it first succeeds, so
// the backend starts (and password login works) while the provider is down.
type OIDCHandler struct {
	cfg      OIDCConfig
	sessions *Handler
	logger   *slog.Logger
	client   *http.Client
	stateKey []byte

	mu       sync.Mutex
	provider *oidc.Provider
}

type oidcState struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Expires  int64  `json:"e"`
}

// NewOIDCHandler builds the OIDC login handler. cfg may be nil, in which case
// only Status is meaningful and reports OIDC as disabled.
func NewOIDCHandler(cfg *OIDCConfig, h *Handler) *OIDCHandler {
	o := &OIDCHandler{
		sessions: h,
		logger:   h.logger,
		client:   &http.Client{Timeout: oidcProviderTimeout},
	}
	if cfg != nil {
		o.cfg = *cfg
		mac := hmac.New(sha256.New, []byte(cfg.StateSecret))
		mac.Write([]byte("doctorine-oidc-state-v1"))
		o.stateKey = mac.Sum(nil)
	}
	return o
}

func (o *OIDCHandler) enabled() bool {
	return o.cfg.IssuerURL != ""
}

type oidcStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// Status reports whether OIDC login is configured.
func (o *OIDCHandler) Status(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, oidcStatusResponse{Enabled: o.enabled()})
}

// Login starts the authorization-code flow with PKCE and redirects to the
// provider's authorization endpoint.
func (o *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !o.enabled() {
		http.NotFound(w, r)
		return
	}
	provider, err := o.discover(r.Context())
	if err != nil {
		o.logger.Warn("oidc discovery", "err", err)
		o.redirectError(w, r, OIDCErrorUnavailable)
		return
	}
	state, err := randomToken()
	if err != nil {
		o.logger.Error("oidc generate state", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		o.logger.Error("oidc generate nonce", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	verifier := oauth2.GenerateVerifier()
	cookieValue, err := o.signState(oidcState{
		State:    state,
		Nonce:    nonce,
		Verifier: verifier,
		Expires:  time.Now().Add(oidcStateTTL).Unix(),
	})
	if err != nil {
		o.logger.Error("oidc sign state", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	http.SetCookie(w, o.stateCookie(cookieValue, int(oidcStateTTL.Seconds())))
	authURL := o.oauth2Config(provider).AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback completes the flow: it checks the signed state cookie, exchanges
// the code with the PKCE verifier, verifies the ID token and starts a normal
// session for the existing user named by preferred_username.
func (o *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if !o.enabled() {
		http.NotFound(w, r)
		return
	}
	http.SetCookie(w, o.stateCookie("", -1))

	query := r.URL.Query()
	if providerErr := query.Get("error"); providerErr != "" {
		o.logger.Warn("oidc provider returned error", "error", providerErr)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	saved, err := o.readState(r)
	if err != nil {
		o.logger.Warn("oidc state cookie", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	if !constantTimeEqual(query.Get("state"), saved.State) {
		o.logger.Warn("oidc state mismatch")
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	code := query.Get("code")
	if code == "" {
		o.logger.Warn("oidc callback without code")
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}

	provider, err := o.discover(r.Context())
	if err != nil {
		o.logger.Warn("oidc discovery", "err", err)
		o.redirectError(w, r, OIDCErrorUnavailable)
		return
	}
	ctx := oidc.ClientContext(r.Context(), o.client)
	token, err := o.oauth2Config(provider).Exchange(ctx, code, oauth2.VerifierOption(saved.Verifier))
	if err != nil {
		o.logger.Warn("oidc code exchange", "err", err)
		if _, isNetErr := errors.AsType[net.Error](err); isNetErr {
			o.redirectError(w, r, OIDCErrorUnavailable)
			return
		}
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	username, err := o.verifiedUsername(ctx, provider, token, saved.Nonce)
	if err != nil {
		o.logger.Warn("oidc id token", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}

	user, err := o.sessions.users.GetByUsername(r.Context(), username)
	if errors.Is(err, ErrNotFound) || (err == nil && user != nil && user.Username != username) {
		o.logger.Warn("oidc login for unknown user", "username", username)
		o.redirectError(w, r, OIDCErrorNoUser)
		return
	}
	if err != nil || user == nil {
		o.logger.Error("oidc user lookup", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	if _, err := o.sessions.startSession(r.Context(), w, user, false); err != nil {
		o.logger.Error("oidc start session", "err", err)
		o.redirectError(w, r, OIDCErrorFailed)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDCHandler) verifiedUsername(
	ctx context.Context,
	provider *oidc.Provider,
	token *oauth2.Token,
	nonce string,
) (string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", errors.New("token response has no id_token")
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: o.cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return "", fmt.Errorf("verify: %w", err)
	}
	if !constantTimeEqual(idToken.Nonce, nonce) {
		return "", errors.New("nonce mismatch")
	}
	var claims struct {
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("decode claims: %w", err)
	}
	if claims.PreferredUsername == "" {
		return "", errors.New("id token has no preferred_username")
	}
	return claims.PreferredUsername, nil
}

func (o *OIDCHandler) discover(ctx context.Context) (*oidc.Provider, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil {
		return o.provider, nil
	}
	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, o.client), oidcProviderTimeout)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, o.cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	o.provider = provider
	return provider, nil
}

func (o *OIDCHandler) oauth2Config(provider *oidc.Provider) *oauth2.Config {
	endpoint := provider.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInHeader
	return &oauth2.Config{
		ClientID:     o.cfg.ClientID,
		ClientSecret: o.cfg.ClientSecret,
		RedirectURL:  o.cfg.RedirectURL,
		Endpoint:     endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

func (o *OIDCHandler) stateCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     OIDCStateCookieName,
		Value:    value,
		Path:     oidcCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   o.sessions.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (o *OIDCHandler) redirectError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+code, http.StatusFound)
}

func (o *OIDCHandler) signState(state oidcState) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(o.mac(encoded)), nil
}

func (o *OIDCHandler) readState(r *http.Request) (oidcState, error) {
	cookie, err := r.Cookie(OIDCStateCookieName)
	if err != nil {
		return oidcState{}, fmt.Errorf("missing state cookie: %w", err)
	}
	encoded, sig, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return oidcState{}, errInvalidStateCookie
	}
	gotMAC, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(gotMAC, o.mac(encoded)) {
		return oidcState{}, errInvalidStateCookie
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return oidcState{}, errInvalidStateCookie
	}
	var state oidcState
	if err := json.Unmarshal(payload, &state); err != nil {
		return oidcState{}, errInvalidStateCookie
	}
	if time.Now().Unix() > state.Expires {
		return oidcState{}, errors.New("state cookie expired")
	}
	if state.State == "" || state.Nonce == "" || state.Verifier == "" {
		return oidcState{}, errInvalidStateCookie
	}
	return state, nil
}

func (o *OIDCHandler) mac(data string) []byte {
	mac := hmac.New(sha256.New, o.stateKey)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func randomToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
