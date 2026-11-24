//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"
)

func jsonKeys(t *testing.T, name string, raw []byte) string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("%s: %v: %s", name, err, raw)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// Every endpoint that answers with a product answers with the same fields.
func TestEveryProductEndpointAnswersWithTheSameShape(t *testing.T) {
	s := newServer(t)
	a := s.register()
	cat := s.categoryID(a.token)
	sku := uniqueSKU()

	created := s.do(http.MethodPost, "/products", a.token, productBody(cat, sku, 5))
	s.expect(created, http.StatusCreated, "")
	var p product
	created.decode(&p)

	update := productBody(cat, sku, 0)
	delete(update, "stock")
	updated := s.do(http.MethodPut, productPath(p), a.token, update)
	s.expect(updated, http.StatusOK, "")

	stock := s.do(http.MethodPatch, productPath(p)+"/stock", a.token, map[string]int{"stock": 9, "expected_stock": 5})
	s.expect(stock, http.StatusOK, "")

	got := s.do(http.MethodGet, productPath(p), a.token, nil)
	s.expect(got, http.StatusOK, "")

	list := s.do(http.MethodGet, "/products", a.token, nil)
	var page struct {
		Items []json.RawMessage `json:"items"`
	}
	list.decode(&page)
	if len(page.Items) != 1 {
		t.Fatalf("list = %s", list.body)
	}

	form := s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(pngFixture(t, 20, 20)))...)
	s.expect(form, http.StatusCreated, "")

	archived := s.do(http.MethodPost, productPath(p)+"/archive", a.token, nil)
	s.expect(archived, http.StatusOK, "")

	want := jsonKeys(t, "GET /products/{id}", got.body)
	for name, raw := range map[string][]byte{
		"POST /products":              created.body,
		"POST /products multipart":    form.body,
		"PUT /products/{id}":          updated.body,
		"PATCH /products/{id}/stock":  stock.body,
		"GET /products item":          page.Items[0],
		"POST /products/{id}/archive": archived.body,
	} {
		if keys := jsonKeys(t, name, raw); keys != want {
			t.Errorf("%s has the fields\n  %s\nwant\n  %s", name, keys, want)
		}
	}
	for _, derived := range []string{"available", "stock_label"} {
		if strings.Contains(want, derived) {
			t.Errorf("a product has the derived field %q", derived)
		}
	}
}
