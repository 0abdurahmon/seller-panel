package auth

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// LoginGuard counts failed attempts per key. Handler uses it for login, code requests and verification.
type LoginGuard interface {
	Check(key string) (bool, time.Duration)
	Hit(key string)
	Reset(key string)
}

// Handler serves the auth endpoints. It only parses requests and maps errors.
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

// LinkTelegram handles POST /auth/telegram/link for the signed-in seller.
func (h *Handler) LinkTelegram(w http.ResponseWriter, r *http.Request) {
	p, ok := MustPrincipal(w, r)
	if !ok {
		return
	}
	res, err := h.svc.StartTelegramLink(r.Context(), p.SellerID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

// RequestCode handles POST /auth/request-code. It answers the same for every email.
func (h *Handler) RequestCode(w http.ResponseWriter, r *http.Request) {
	var req RequestCodeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	key := "code|" + h.clientIP(r) + "|" + normalizeLookup(req.Email)
	if allowed, wait := h.guard.Check(key); !allowed {
		httpx.WriteRateLimited(w, wait)
		return
	}
	h.guard.Hit(key)
	if err := h.svc.RequestCode(r.Context(), req.Email); err != nil {
		writeAuthError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, CodeSentResponse{
		Status:     "sent",
		RetryAfter: int(math.Ceil(h.svc.CodeCooldown().Seconds())),
	})
}

// Verify handles POST /auth/verify, which trades a sign-in code for a token.
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	key := "verify|" + h.clientIP(r) + "|" + normalizeLookup(req.Email)
	if allowed, wait := h.guard.Check(key); !allowed {
		httpx.WriteRateLimited(w, wait)
		return
	}
	res, err := h.svc.VerifyCode(r.Context(), req.Email, req.Code)
	if errors.Is(err, ErrInvalidCode) {
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
	var cooldown *CooldownError
	if errors.As(err, &cooldown) {
		httpx.WriteRateLimited(w, cooldown.RetryAfter)
		return
	}
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "this email is already registered")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, ErrInvalidCode):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_code", "the code is wrong or has expired")
	case errors.Is(err, ErrTelegramNotLinked):
		httpx.WriteError(w, http.StatusConflict, "telegram_not_linked", "no Telegram chat is linked to this account, so no code can be sent")
	case errors.Is(err, ErrTelegramUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "telegram_unavailable", "sign-in by Telegram code is not set up on this server")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
