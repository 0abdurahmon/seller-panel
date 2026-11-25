package order

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/realtime"
)

// streamRetryAfter is how long a client waits before opening another stream once the shop has too many.
const streamRetryAfter = 5 * time.Second

// Handler serves the order endpoints. The shop comes from the token only.
type Handler struct {
	svc *Service
	hub *realtime.Hub
}

// NewHandler returns the HTTP handler for orders. hub serves GET /orders/events.
func NewHandler(svc *Service, hub *realtime.Hub) *Handler {
	return &Handler{svc: svc, hub: hub}
}

// Events handles GET /orders/events, a stream of order.created and order.status_changed for the shop.
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	sub, err := h.hub.Subscribe(p.ShopID)
	if errors.Is(err, realtime.ErrTooManyStreams) {
		httpx.WriteRateLimited(w, streamRetryAfter)
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
		return
	}
	defer sub.Close()
	realtime.Stream(w, r, sub, p.Expires)
}

// List handles GET /orders?status=&page=&page_size=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	var status *Status
	if raw := r.URL.Query().Get("status"); raw != "" {
		s := Status(raw)
		status = &s
	}
	page, err := intParam(r, "page", 1)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "page must be an integer")
		return
	}
	pageSize, err := intParam(r, "page_size", 0)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "page_size must be an integer")
		return
	}
	res, err := h.svc.List(r.Context(), p.ShopID, status, page, pageSize)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// Get handles GET /orders/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "order not found")
		return
	}
	o, err := h.svc.Get(r.Context(), p.ShopID, id)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

// Simulate handles POST /dev/simulate-order: a fake buyer order from the shop's own products.
func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	o, err := h.svc.Simulate(r.Context(), p.ShopID)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, withLists(o))
}

// Transition handles POST /orders/{id}/transition.
func (h *Handler) Transition(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "order not found")
		return
	}
	var req TransitionRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	o, err := h.svc.Transition(r.Context(), p.ShopID, id, req)
	if err != nil {
		writeOrderError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

func intParam(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func writeOrderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "order not found")
	case errors.Is(err, ErrInvalidTransition):
		httpx.WriteError(w, http.StatusConflict, "invalid_transition", "this status change is not allowed")
	case errors.Is(err, ErrInsufficientStock):
		httpx.WriteError(w, http.StatusConflict, "insufficient_stock", "not enough stock for this order")
	case errors.Is(err, ErrNoProducts):
		httpx.WriteError(w, http.StatusConflict, "no_products_to_order", "the shop has no active product with available stock")
	case errors.Is(err, ErrProductNotFound):
		httpx.WriteError(w, http.StatusConflict, "product_not_found", "a product in this order no longer exists")
	default:
		slog.ErrorContext(r.Context(), "order request failed", "path", r.URL.Path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
