package upload

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

const (
	formField = "file"
	// multipartOverhead leaves room for boundaries and headers around a file of MaxFileBytes.
	multipartOverhead = 64 << 10
)

// Handler serves POST /uploads. The files themselves are served by nginx from the uploads volume.
type Handler struct {
	svc *Service
}

// NewHandler returns the HTTP handler for uploads.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /uploads, a multipart form with one image in the field "file".
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.MustPrincipal(w, r)
	if !ok {
		return
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be multipart/form-data")
		return
	}
	if r.ContentLength > MaxFileBytes+multipartOverhead {
		writeUploadError(w, r, ErrTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileBytes+multipartOverhead)
	mr, err := r.MultipartReader()
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is not a valid multipart form")
		return
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			if isTooLarge(err) {
				writeUploadError(w, r, ErrTooLarge)
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", `form field "file" is required`)
			return
		}
		if part.FormName() != formField || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		img, err := h.svc.Upload(r.Context(), p.ShopID, part)
		if err != nil {
			if isTooLarge(err) {
				err = ErrTooLarge
			}
			writeUploadError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, img)
		return
	}
}

func isTooLarge(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

func writeUploadError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "file is larger than 5 MB")
	case errors.Is(err, ErrUnsupportedType):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "file not found")
	case errors.Is(err, ErrBusy):
		w.Header().Set("Retry-After", "2")
		httpx.WriteError(w, http.StatusServiceUnavailable, "busy", "server is busy processing images, try again shortly")
	case errors.Is(err, r.Context().Err()) && r.Context().Err() != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "busy", "request cancelled")
	default:
		slog.Error("upload failed", "error", err, "path", r.URL.Path)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
