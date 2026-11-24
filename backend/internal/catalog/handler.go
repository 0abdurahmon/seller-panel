package catalog

import (
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}
