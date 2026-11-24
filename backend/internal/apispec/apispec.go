package apispec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type Handler struct {
	path string

	mu      sync.Mutex
	modTime time.Time
	size    int64
	body    []byte
}

func NewHandler(path string) *Handler {
	return &Handler{path: path}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := h.json()
	switch {
	case errors.Is(err, fs.ErrNotExist):
		httpx.WriteError(w, http.StatusNotFound, "spec_not_found", "the OpenAPI file is not available")
		return
	case err != nil:
		slog.ErrorContext(r.Context(), "serve openapi failed", "file", h.path, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}

func (h *Handler) json() ([]byte, error) {
	info, err := os.Stat(h.path)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.body != nil && info.ModTime().Equal(h.modTime) && info.Size() == h.size {
		return h.body, nil
	}

	raw, err := os.ReadFile(h.path)
	if err != nil {
		return nil, err
	}
	compact, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("convert %s to JSON: %w", h.path, err)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, compact, "", "  "); err != nil {
		return nil, fmt.Errorf("indent JSON: %w", err)
	}
	pretty.WriteByte('\n')

	h.body, h.modTime, h.size = pretty.Bytes(), info.ModTime(), info.Size()
	return h.body, nil
}
