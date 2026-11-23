package stock

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// UpdateRequest: new absolute stock and the stock seen before editing; both required.
type UpdateRequest struct {
	Stock         *int `json:"stock"`
	ExpectedStock *int `json:"expected_stock"`
}

// ConflictResponse is a 409 error body that also carries the current stock.
type ConflictResponse struct {
	httpx.Error
	Current Level `json:"current"`
}

// Handler serves the stock endpoints. It only parses requests and maps errors.
type Handler struct {
	svc *Service
}

// NewHandler returns the HTTP handler for stock.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Update handles PATCH /products/{id}/stock behind auth; shop comes from the token.
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
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", `request body must be JSON like {"stock": 12, "expected_stock": 10}`)
		return
	}
	if req.Stock == nil || req.ExpectedStock == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "stock and expected_stock are required")
		return
	}
	level, err := h.svc.SetStock(r.Context(), p.ShopID, productID, *req.Stock, *req.ExpectedStock)
	if err != nil {
		writeStockError(w, r, level, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, level)
}

func writeStockError(w http.ResponseWriter, r *http.Request, current Level, err error) {
	conflict := func(code, message string) {
		httpx.WriteJSON(w, http.StatusConflict, ConflictResponse{
			Error:   httpx.Error{Code: code, Message: message},
			Current: current,
		})
	}
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
	case errors.Is(err, ErrArchived):
		httpx.WriteError(w, http.StatusConflict, "product_archived", "an archived product's stock cannot be changed")
	case errors.Is(err, ErrStockChanged):
		conflict("stock_changed", "stock was changed since you opened it; check the current value and try again")
	case errors.Is(err, ErrBelowReserved):
		conflict("stock_below_reserved", "stock cannot be lower than the units reserved by accepted orders")
	default:
		slog.ErrorContext(r.Context(), "set stock failed", "path", r.URL.Path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
