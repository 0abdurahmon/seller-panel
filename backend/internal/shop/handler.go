package shop

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// Handler serves the shop endpoints. The shop ID always comes from the
// authenticated principal, never from the request.
type Handler struct {
	svc *Service
}

// NewHandler returns the HTTP handler for the shop profile.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Get handles GET /shop.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	s, err := h.svc.Get(r.Context(), p.ShopID)
	if err != nil {
		writeShopError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

// Update handles PUT /shop.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	var lc LogoChange
	if httpx.IsMultipart(r) {
		var err error
		if req, lc, err = readForm(w, r); err != nil {
			writeShopError(w, r, err)
			return
		}
	} else if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	s, err := h.svc.Update(r.Context(), p.ShopID, req, lc)
	if err != nil {
		writeShopError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func writeShopError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errInvalidBody):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not a valid multipart form")
	case errors.Is(err, ErrFileTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "the logo is larger than 5 MB")
	case errors.Is(err, ErrUnsupportedMedia):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
	case errors.Is(err, ErrBusy):
		w.Header().Set("Retry-After", "2")
		httpx.WriteError(w, http.StatusServiceUnavailable, "busy", "server is busy processing images, try again shortly")
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "shop not found")
	default:
		slog.ErrorContext(r.Context(), "shop request failed", "path", r.URL.Path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
