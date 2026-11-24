package shop

import (
	"errors"
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
		writeShopError(w, err)
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
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	s, err := h.svc.Update(r.Context(), p.ShopID, req)
	if err != nil {
		writeShopError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func writeShopError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "shop not found")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
