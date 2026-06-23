package auth

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const (
	sessionTTL  = 24 * time.Hour
	rememberTTL = 5 * 24 * time.Hour
)

type Handler struct {
	users        UserStore
	sessions     SessionStore
	tokens       PersonalTokenStore
	logger       *slog.Logger
	cookieSecure bool
}

func NewHandler(store *Store, cookieSecure bool, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		users:        store,
		sessions:     store,
		tokens:       store,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID          int     `json:"id"`
	Username    string  `json:"username"`
	IsAdmin     bool    `json:"is_admin"`
	DisplayName *string `json:"display_name"`
	CreatedAt   string  `json:"created_at"`
}

func toUserResponse(user *User) userResponse {
	return userResponse{
		ID:          user.ID,
		Username:    user.Username,
		IsAdmin:     user.IsAdmin,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.users.GetByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || user == nil || !CheckPassword(user.PasswordHash, req.Password) {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	ttl := sessionTTL
	if req.RememberMe {
		ttl = rememberTTL
	}
	token, tokenHash, err := GenerateSessionToken()
	if err != nil {
		h.logger.Error("generate session token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	expiresAt := time.Now().UTC().Add(ttl)
	if err := h.sessions.CreateSession(r.Context(), user.ID, tokenHash, expiresAt); err != nil {
		h.logger.Error("create session", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.setSessionCookie(w, token, req.RememberMe, ttl)
	httputil.WriteJSON(w, http.StatusOK, loginResponse{Token: token, User: toUserResponse(user)})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := tokenFromRequest(r); token != "" {
		if err := h.sessions.RevokeSession(r.Context(), HashSessionToken(token)); err != nil {
			h.logger.Warn("revoke session", "err", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, remember bool, ttl time.Duration) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	if remember {
		cookie.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, cookie)
}
