package product

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type StockLabeler func(available int) string

type Response struct {
	Product
	Available  int    `json:"available"`
	StockLabel string `json:"stock_label"`
}

type PageResponse struct {
	Items    []Response `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

type Handler struct {
	svc   *Service
	label StockLabeler
}

func NewHandler(svc *Service, label StockLabeler) *Handler {
	return &Handler{svc: svc, label: label}
}

func (h *Handler) response(p Product) Response {
	if p.Photos == nil {
		p.Photos = []string{}
	}
	available := p.Stock - p.Reserved
	return Response{Product: p, Available: available, StockLabel: h.label(available)}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := ListFilter{Query: q.Get("q"), Status: Status(q.Get("status"))}
	var err error
	if f.CategoryID, err = queryInt64(q.Get("category"), "category"); err != nil {
		writeError(w, err)
		return
	}
	page, err := queryInt64(q.Get("page"), "page")
	if err != nil {
		writeError(w, err)
		return
	}
	pageSize, err := queryInt64(q.Get("page_size"), "page_size")
	if err != nil {
		writeError(w, err)
		return
	}
	f.Page, f.PageSize = int(page), int(pageSize)

	res, err := h.svc.List(r.Context(), p.ShopID, f)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]Response, len(res.Items))
	for i, it := range res.Items {
		items[i] = h.response(it)
	}
	httpx.WriteJSON(w, http.StatusOK, PageResponse{Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	var in CreateInput
	var add [][]byte
	if isMultipart(r) {
		f, err := readForm(w, r, createFields)
		if err == nil {
			in, err = f.createInput()
		}
		if err != nil {
			writeError(w, err)
			return
		}
		add = f.files
	} else if err := httpx.DecodeJSON(w, r, &in); err != nil {
		writeInvalidBody(w)
		return
	}
	res, err := h.svc.Create(r.Context(), p.ShopID, in, add)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, h.response(res))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Get(r.Context(), p.ShopID, pathID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.response(res))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	var in UpdateInput
	var pc PhotoChange
	if isMultipart(r) {
		f, err := readForm(w, r, updateFields)
		if err == nil {
			in, err = f.updateInput()
		}
		if err != nil {
			writeError(w, err)
			return
		}
		pc = PhotoChange{Keep: f.keep, Add: f.files}
	} else if err := httpx.DecodeJSON(w, r, &in); err != nil {
		writeInvalidBody(w)
		return
	}
	res, err := h.svc.Update(r.Context(), p.ShopID, pathID(r), in, pc)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.response(res))
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Archive(r.Context(), p.ShopID, pathID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.response(res))
}

func pathID(r *http.Request) int64 {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func queryInt64(raw, name string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, invalid("%s must be an integer", name)
	}
	return n, nil
}

func writeInvalidBody(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
}

var errInvalidBody = errors.New("request body is not valid")

func isTooLarge(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidBody):
		writeInvalidBody(w)
	case errors.Is(err, ErrFileTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "a photo is larger than 5 MB")
	case errors.Is(err, ErrUnsupportedMedia):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
	case errors.Is(err, ErrBusy):
		w.Header().Set("Retry-After", "2")
		httpx.WriteError(w, http.StatusServiceUnavailable, "busy", "server is busy processing images, try again shortly")
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "product not found")
	case errors.Is(err, ErrSKUTaken):
		httpx.WriteError(w, http.StatusConflict, "sku_taken", "this sku is already used in your shop")
	case errors.Is(err, ErrArchived):
		httpx.WriteError(w, http.StatusConflict, "product_archived", "archived products cannot be edited")
	default:
		slog.Error("product request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
