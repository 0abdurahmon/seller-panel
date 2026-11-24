package stock

import (
	"errors"
	"log/slog"
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

// Update handles PATCH /products/{id}/stock; the shop comes from the token.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "product not found")
		return
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", `request body must be JSON like {"stock": 12, "expected_stock": 10}`)
		return
	}
	if req.Stock == nil || req.ExpectedStock == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "stock and expected_stock are required")
		return
	}
	level, err := h.svc.SetStock(r.Context(), p.ShopID, productID, *req.Stock, *req.ExpectedStock)
	if err != nil {
		writeStockError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, level)
}

func writeStockError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "product not found")
	case errors.Is(err, ErrArchived):
		httpx.WriteError(w, http.StatusConflict, "product_archived", "an archived product's stock cannot be changed")
	case errors.Is(err, ErrStockChanged):
		httpx.WriteError(w, http.StatusConflict, "stock_changed", "stock was changed since you opened it; reload and try again")
	case errors.Is(err, ErrBelowReserved):
		httpx.WriteError(w, http.StatusConflict, "stock_below_reserved", "stock cannot be lower than the units reserved by accepted orders")
	default:
		slog.ErrorContext(r.Context(), "set stock failed", "path", r.URL.Path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
