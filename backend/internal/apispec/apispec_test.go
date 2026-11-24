package apispec

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const sampleSpec = `openapi: 3.0.3
info:
  title: Sample
paths:
  /things:
    get:
      responses:
        200:
          description: unquoted integer key
        '404':
          description: quoted key
`

func writeSpec(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func get(h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, rec.Body.String())
	}
	return doc
}

func TestServesTheYAMLAsJSON(t *testing.T) {
	rec := get(NewHandler(writeSpec(t, sampleSpec)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store so an edited spec is never served stale", cc)
	}
	if !strings.HasPrefix(rec.Body.String(), "{\n  \"") || !strings.HasSuffix(rec.Body.String(), "}\n") {
		t.Fatalf("JSON is not indented for reading: %q", rec.Body.String())
	}

	doc := decode(t, rec)
	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi = %v", doc["openapi"])
	}
	responses := doc["paths"].(map[string]any)["/things"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)
	if _, ok := responses["200"]; !ok {
		t.Fatalf("an unquoted integer key must become the string \"200\", got %v", responses)
	}
	if _, ok := responses["404"]; !ok {
		t.Fatalf("responses = %v", responses)
	}
}

func TestMissingFileIsAJSON404(t *testing.T) {
	rec := get(NewHandler(filepath.Join(t.TempDir(), "nope.yaml")))
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || rec.Code != http.StatusNotFound || e.Code != "spec_not_found" || e.Message == "" {
		t.Fatalf("status %d body %s, want 404 spec_not_found", rec.Code, rec.Body.String())
	}
}

func TestInvalidYAMLIs500AndHidesTheDetails(t *testing.T) {
	rec := get(NewHandler(writeSpec(t, "openapi: 3.0.3\npaths: [\n")))
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || rec.Code != http.StatusInternalServerError || e.Code != "internal" {
		t.Fatalf("status %d body %s, want 500 internal", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "yaml") || strings.Contains(rec.Body.String(), "openapi.yaml") {
		t.Fatalf("the error leaks internals: %s", rec.Body.String())
	}
}

func TestFollowsEditsWithoutARestart(t *testing.T) {
	path := writeSpec(t, sampleSpec)
	h := NewHandler(path)

	first := get(h)
	if again := get(h); again.Body.String() != first.Body.String() {
		t.Fatal("an unchanged file must give the same JSON")
	}

	edited := strings.Replace(sampleSpec, "title: Sample", "title: Edited", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	doc := decode(t, get(h))
	if title := doc["info"].(map[string]any)["title"]; title != "Edited" {
		t.Fatalf("title = %v after the edit, want Edited", title)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if rec := get(h); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d after the file was removed, want 404", rec.Code)
	}
}

func TestSafeForConcurrentRequestsAndEdits(t *testing.T) {
	path := writeSpec(t, sampleSpec)
	h := NewHandler(path)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if rec := get(h); rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
					t.Errorf("status = %d", rec.Code)
					return
				}
			}
		}()
	}
	for j := 0; j < 20; j++ {
		stamp := time.Now().Add(time.Duration(j+1) * time.Minute)
		_ = os.Chtimes(path, stamp, stamp)
	}
	wg.Wait()
}

func TestRepositorySpecConverts(t *testing.T) {
	rec := get(NewHandler(filepath.Join("..", "..", "..", "api", "openapi.yaml")))
	if rec.Code != http.StatusOK {
		t.Fatalf("api/openapi.yaml does not convert: %d %s", rec.Code, rec.Body.String())
	}
	doc := decode(t, rec)
	if doc["openapi"] == nil || doc["info"] == nil {
		t.Fatalf("missing top-level keys: %v", doc)
	}
	paths, _ := doc["paths"].(map[string]any)
	if len(paths) < 10 {
		t.Fatalf("only %d paths, the spec looks truncated", len(paths))
	}
	for path, item := range paths {
		if len(item.(map[string]any)) == 0 {
			t.Errorf("%s has no operations", path)
		}
	}
}
