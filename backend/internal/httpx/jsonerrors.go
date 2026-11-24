package httpx

import (
	"net/http"
	"strings"
)

type jsonErrorWriter struct {
	http.ResponseWriter
	wroteHeader bool
	replaced    bool
}

func JSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&jsonErrorWriter{ResponseWriter: w}, r)
	})
}

func (w *jsonErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *jsonErrorWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if isPlainError(status, w.Header().Get("Content-Type")) {
		w.replaced = true
		code, message := "not_found", "route not found"
		if status == http.StatusMethodNotAllowed {
			code, message = "method_not_allowed", "method not allowed for this route"
		}
		w.Header().Del("Content-Length")
		WriteError(w.ResponseWriter, status, code, message)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *jsonErrorWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.replaced {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

func isPlainError(status int, contentType string) bool {
	if status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
		return false
	}
	return !strings.HasPrefix(contentType, "application/json")
}
