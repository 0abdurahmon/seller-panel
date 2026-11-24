package balance

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Summary handles GET /balance.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Summary(r.Context(), p.ShopID)
	if err != nil {
		writeBalanceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// Entries handles GET /balance/entries.
func (h *Handler) Entries(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := queryInt(q.Get("page"), "page")
	if err != nil {
		writeBalanceError(w, r, err)
		return
	}
	pageSize, err := queryInt(q.Get("page_size"), "page_size")
	if err != nil {
		writeBalanceError(w, r, err)
		return
	}
	res, err := h.svc.Entries(r.Context(), p.ShopID, page, pageSize)
	if err != nil {
		writeBalanceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func queryInt(raw, name string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be an integer", ErrValidation, name)
	}
	return n, nil
}

func writeBalanceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		slog.ErrorContext(r.Context(), "balance request failed", "path", r.URL.Path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
