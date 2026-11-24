package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

func newTestMux(t *testing.T, repo *memRepository) (http.Handler, string) {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(repo))
	mux := http.NewServeMux()
	mux.Handle("GET /categories", auth.RequireAuth(tokens)(http.HandlerFunc(h.List)))
	token, _, err := tokens.Issue(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	return mux, token
}

func TestHandlerListReturnsItems(t *testing.T) {
	parent := int64(1)
	mux, token := newTestMux(t, &memRepository{items: []Category{
		{ID: 1, Name: "Electronics"},
		{ID: 2, Name: "Phones", ParentID: &parent},
	}})

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	items := got["items"]
	if len(items) != 2 || items[0]["parent_id"] != nil || items[1]["parent_id"] != float64(1) {
		t.Fatalf("body = %v", got)
	}
}

func TestHandlerListEmptyIsArray(t *testing.T) {
	mux, token := newTestMux(t, &memRepository{})
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if body := rec.Body.String(); body != "{\"items\":[]}\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestHandlerListRequiresToken(t *testing.T) {
	mux, _ := newTestMux(t, &memRepository{})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/categories", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerListMapsErrorTo500(t *testing.T) {
	mux, token := newTestMux(t, &memRepository{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var e struct{ Code, Message string }
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil || e.Code != "internal" || e.Message == "db down" {
		t.Fatalf("error body = %+v, %v", e, err)
	}
}
