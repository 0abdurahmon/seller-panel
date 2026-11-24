package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// Handler serves the auth endpoints. It only parses requests and maps errors.
type LoginGuard interface {
	Check(key string) (bool, time.Duration)
	Hit(key string)
	Reset(key string)
}

type Handler struct {
	svc      *Service
	guard    LoginGuard
	clientIP func(*http.Request) string
}

// NewHandler returns the HTTP handler for auth.
func NewHandler(svc *Service, guard LoginGuard, clientIP func(*http.Request) string) *Handler {
	return &Handler{svc: svc, guard: guard, clientIP: clientIP}
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	key := h.clientIP(r) + "|" + strings.ToLower(strings.TrimSpace(req.Email))
	if allowed, wait := h.guard.Check(key); !allowed {
		httpx.WriteRateLimited(w, wait)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if errors.Is(err, ErrInvalidCredentials) {
		h.guard.Hit(key)
	}
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.guard.Reset(key)
	httpx.WriteJSON(w, http.StatusOK, res)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "this email is already registered")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
