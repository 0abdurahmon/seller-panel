package auth

import (
	"errors"
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// Handler serves the auth endpoints. It only parses requests and maps errors.
type Handler struct {
	svc *Service
}

// NewHandler returns the HTTP handler for auth.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
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
