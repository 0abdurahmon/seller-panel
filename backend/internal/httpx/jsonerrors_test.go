package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /gone", func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, http.StatusNotFound, "product_missing", "no such product")
	})
	mux.HandleFunc("GET /plain", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "plain text failure", http.StatusInternalServerError)
	})
	mux.HandleFunc("GET /teapot", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short and stout"))
	})
	mux.HandleFunc("GET /implicit", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})
	return JSONErrors(mux)
}

func serve(method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newTestMux().ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) Error {
	t.Helper()
	var e Error
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
	return e
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	rec := serve(http.MethodGet, "/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}
	if e := errorBody(t, rec); e.Code != "not_found" || e.Message == "" {
		t.Fatalf("error = %+v", e)
	}
	if strings.Contains(rec.Body.String(), "page not found") {
		t.Fatalf("the default text leaked into the body: %q", rec.Body.String())
	}
}

func TestWrongMethodIsJSON405WithAllow(t *testing.T) {
	rec := serve(http.MethodPost, "/ok")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow = %q, want it kept", allow)
	}
	if e := errorBody(t, rec); e.Code != "method_not_allowed" || e.Message == "" {
		t.Fatalf("error = %+v", e)
	}
}

func TestHandlerOwnJSON404PassesThrough(t *testing.T) {
	rec := serve(http.MethodGet, "/gone")
	if e := errorBody(t, rec); rec.Code != http.StatusNotFound || e.Code != "product_missing" || e.Message != "no such product" {
		t.Fatalf("status %d error %+v, the handler's own error must stay", rec.Code, e)
	}
}

func TestOtherResponsesAreUntouched(t *testing.T) {
	if rec := serve(http.MethodGet, "/ok"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("200: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := serve(http.MethodGet, "/plain"); rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "plain text failure") {
		t.Fatalf("500 text: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := serve(http.MethodGet, "/teapot"); rec.Code != http.StatusTeapot || rec.Body.String() != "short and stout" {
		t.Fatalf("418: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := serve(http.MethodGet, "/implicit"); rec.Code != http.StatusOK || rec.Body.String() != "hello" {
		t.Fatalf("implicit 200: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestHeadRequestToUnknownRoute(t *testing.T) {
	rec := serve(http.MethodHead, "/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStatusIsWrittenOnce(t *testing.T) {
	h := JSONErrors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("late"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want the first one (404)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "late") {
		t.Fatalf("body after the replaced error leaked: %q", rec.Body.String())
	}
}
