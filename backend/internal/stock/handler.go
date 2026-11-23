package stock

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// Handler serves the stock endpoints. It only parses requests and maps errors.
type Handler struct {
	svc *Service
}

// NewHandler returns the HTTP handler for stock.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Update handles PATCH /products/{id}/stock. It must run behind
// auth.RequireAuth: the shop comes from the token, never from the request.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return
	}
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be JSON like {\"stock\": 10}")
		return
	}
	if req.Stock == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "stock is required")
		return
	}
	level, err := h.svc.SetStock(r.Context(), p.ShopID, productID, *req.Stock)
	if err != nil {
		writeStockError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, level)
}

func writeStockError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
	case errors.Is(err, ErrArchived):
		httpx.WriteError(w, http.StatusConflict, "product_archived", "an archived product's stock cannot be changed")
	case errors.Is(err, ErrBelowReserved):
		httpx.WriteError(w, http.StatusConflict, "stock_below_reserved", "stock cannot be lower than the units reserved by accepted orders")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
