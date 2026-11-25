//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

func TestOpenAPIJSONDescribesTheImplementedAPI(t *testing.T) {
	s := newServer(t)

	res := s.do(http.MethodGet, "/openapi.json", "", nil)
	s.expect(res, http.StatusOK, "")
	if ct := res.header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	var doc struct {
		OpenAPI string                           `json:"openapi"`
		Servers []struct{ URL string }           `json:"servers"`
		Paths   map[string]map[string]any        `json:"paths"`
		Comp    struct{ Schemas map[string]any } `json:"components"`
	}
	res.decode(&doc)
	if !strings.HasPrefix(doc.OpenAPI, "3.") || len(doc.Servers) == 0 || doc.Servers[0].URL != "/api" {
		t.Fatalf("openapi %q, servers %+v", doc.OpenAPI, doc.Servers)
	}

	for _, op := range []struct{ path, method string }{
		{"/auth/register", "post"}, {"/auth/login", "post"}, {"/shop", "put"}, {"/categories", "get"},
		{"/products", "post"}, {"/products/{id}/archive", "post"}, {"/products/{id}/stock", "patch"},
		{"/orders", "get"}, {"/orders/{id}/transition", "post"}, {"/balance", "get"}, {"/balance/entries", "get"},
		{"/dev/simulate-order", "post"}, {"/auth/request-code", "post"}, {"/auth/verify", "post"},
		{"/auth/telegram/link", "post"}, {"/orders/events", "get"},
		{"/dev/simulate-order", "post"},
		{"/auth/request-code", "post"}, {"/auth/verify", "post"}, {"/auth/telegram/link", "post"},
	} {
		if _, ok := doc.Paths[op.path][op.method]; !ok {
			t.Errorf("the spec has no %s %s", strings.ToUpper(op.method), op.path)
		}
	}
	if _, ok := doc.Paths["/uploads"]; ok {
		t.Error("the spec still has /uploads, images are saved by the product and shop endpoints")
	}
	for _, path := range []string{"/products", "/orders", "/balance/entries"} {
		get, _ := doc.Paths[path]["get"].(map[string]any)
		responses, _ := get["responses"].(map[string]any)
		if _, ok := responses["400"]; !ok {
			t.Errorf("the spec has no 400 response for GET %s, a bad query is refused", path)
		}
	}
	for _, name := range []string{"Product", "Order", "OrderHistoryEntry", "Error", "TokenResponse"} {
		if _, ok := doc.Comp.Schemas[name]; !ok {
			t.Errorf("the spec has no schema %s", name)
		}
	}
}

func TestOpenAPIJSONIsNotAvailableWithoutTheFile(t *testing.T) {
	missing := newServer(t, func(c *config.Config) { c.OpenAPIFile = "/nonexistent/openapi.yaml" })
	missing.expect(missing.do(http.MethodGet, "/openapi.json", "", nil), http.StatusNotFound, "spec_not_found")

	off := newServer(t, func(c *config.Config) { c.OpenAPIFile = "" })
	off.expect(off.do(http.MethodGet, "/openapi.json", "", nil), http.StatusNotFound, "not_found")
}
