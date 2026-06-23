package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const maxTokenName = 120

type createTokenRequest struct {
	Name      string  `json:"name"`
	Scope     string  `json:"scope"`
	ExpiresAt *string `json:"expires_at"`
}

type tokenResponse struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Scope      string  `json:"scope"`
	ExpiresAt  *string `json:"expires_at"`
	LastUsedAt *string `json:"last_used_at"`
	CreatedAt  string  `json:"created_at"`
}

type createTokenResponse struct {
	tokenResponse
	// Token is the raw secret, returned only once at creation.
	Token string `json:"token"`
}

func toTokenResponse(token PersonalToken) tokenResponse {
	return tokenResponse{
		ID:         token.ID,
		Name:       token.Name,
		Scope:      token.Scope,
		ExpiresAt:  formatOptionalTime(token.ExpiresAt),
		LastUsedAt: formatOptionalTime(token.LastUsedAt),
		CreatedAt:  token.CreatedAt.Format(time.RFC3339),
	}
}

// CreateToken issues a new long-lived personal access token for the current
// user. The raw token is returned once and never stored in clear text.
func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	var req createTokenRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "Name is required")
		return
	}
	if len(name) > maxTokenName {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "Name is too long")
		return
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = ScopeFull
	}
	if scope != ScopeFull && scope != ScopeRead {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "Scope must be full or read")
		return
	}
	expiresAt, detail := parseTokenExpiry(req.ExpiresAt)
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, detail)
		return
	}

	raw, hash, err := GeneratePersonalToken()
	if err != nil {
		h.logger.Error("generate personal token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	token, err := h.tokens.CreatePersonalToken(r.Context(), user.ID, hash, name, scope, expiresAt)
	if err != nil {
		h.logger.Error("create personal token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, createTokenResponse{
		tokenResponse: toTokenResponse(token),
		Token:         raw,
	})
}

// ListTokens returns the current user's non-revoked tokens. Expired (but not
// revoked) tokens are still listed so the owner can see and remove them; the UI
// flags them as expired.
func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tokens, err := h.tokens.ListPersonalTokens(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list personal tokens", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]tokenResponse, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, toTokenResponse(token))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// RevokeToken revokes a token the current user owns.
func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.tokens.RevokePersonalToken(r.Context(), user.ID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Token not found")
			return
		}
		h.logger.Error("revoke personal token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseTokenExpiry accepts an RFC3339 datetime or a YYYY-MM-DD date (interpreted
// as end of that day, UTC). A nil/empty value means the token never expires.
func parseTokenExpiry(raw *string) (*time.Time, string) {
	if raw == nil {
		return nil, ""
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		utc := parsed.UTC()
		return validateFutureExpiry(utc)
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		utc := parsed.UTC().Add(24*time.Hour - time.Second)
		return validateFutureExpiry(utc)
	}
	return nil, "Expiry must use RFC3339 or YYYY-MM-DD"
}

func validateFutureExpiry(t time.Time) (*time.Time, string) {
	if !t.After(time.Now().UTC()) {
		return nil, "Expiry must be in the future"
	}
	return &t, ""
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
