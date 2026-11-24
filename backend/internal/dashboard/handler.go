package dashboard

import (
	"log/slog"
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Summary handles GET /dashboard/summary.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Summary(r.Context(), p.ShopID)
	if err != nil {
		writeDashboardError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func writeDashboardError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "dashboard request failed", "path", r.URL.Path, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
}
