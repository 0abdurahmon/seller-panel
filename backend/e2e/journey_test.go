//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestSellerJourney(t *testing.T) {
	s := newServer(t)
	sel := s.register()

	login := s.login(sel.email, sel.password)
	s.expect(login, http.StatusOK, "")
	var tok struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	login.decode(&tok)
	if tok.Token == "" || tok.ExpiresAt == "" {
		t.Fatalf("login must return a token and its expiry: %s", login.body)
	}
	token := tok.Token

	shop := s.do(http.MethodGet, "/shop", token, nil)
	s.expect(shop, http.StatusOK, "")
	if !strings.Contains(string(shop.body), sel.shopName) {
		t.Fatalf("shop = %s, want the name %q chosen at sign up", shop.body, sel.shopName)
	}
	update := s.do(http.MethodPut, "/shop", token, map[string]string{
		"name": "Renamed shop", "description": "We sell phones", "logo_url": "",
		"contact_phone": "+998901234567", "pickup_address": "Tashkent",
	})
	s.expect(update, http.StatusOK, "")
	if again := s.do(http.MethodGet, "/shop", token, nil); !strings.Contains(string(again.body), "Renamed shop") {
		t.Fatalf("shop after update = %s", again.body)
	}

	category := s.categoryID(token)
	sku := uniqueSKU()
	created := s.do(http.MethodPost, "/products", token, productBody(category, sku, 3))
	s.expect(created, http.StatusCreated, "")
	var p product
	created.decode(&p)
	if p.ID == 0 || p.Stock != 3 || p.Reserved != 0 || p.Available != 3 || p.StockLabel != "low_stock" || p.Status != "active" {
		t.Fatalf("created product = %+v", p)
	}

	var page productPage
	s.do(http.MethodGet, "/products?q="+strings.ToLower(sku), token, nil).decode(&page)
	if page.Total != 1 || page.Items[0].ID != p.ID {
		t.Fatalf("search by sku = %+v", page)
	}
	s.do(http.MethodGet, "/products?status=draft", token, nil).decode(&page)
	if page.Total != 0 || page.Items == nil {
		t.Fatalf("draft filter = %+v, want an empty page with items []", page)
	}

	renamed := productBody(category, sku, 0)
	delete(renamed, "stock")
	renamed["title"], renamed["price"] = "Phone Pro", 2_000_000
	res := s.do(http.MethodPut, productPath(p), token, renamed)
	s.expect(res, http.StatusOK, "")
	res.decode(&p)
	if p.Title != "Phone Pro" || p.Price != 2_000_000 || p.Stock != 3 {
		t.Fatalf("after update = %+v, the stock must stay 3", p)
	}

	res = s.do(http.MethodPatch, productPath(p)+"/stock", token, map[string]int{"stock": 50, "expected_stock": 3})
	s.expect(res, http.StatusOK, "")
	var level struct {
		ProductID  int64  `json:"product_id"`
		Stock      int    `json:"stock"`
		Available  int    `json:"available"`
		StockLabel string `json:"stock_label"`
	}
	res.decode(&level)
	if level.ProductID != p.ID || level.Stock != 50 || level.Available != 50 || level.StockLabel != "in_stock" {
		t.Fatalf("stock level = %+v", level)
	}
	s.do(http.MethodGet, productPath(p), token, nil).decode(&p)
	if p.Stock != 50 || p.StockLabel != "in_stock" {
		t.Fatalf("product after the stock change = %+v", p)
	}

	for i := 0; i < 2; i++ {
		res = s.do(http.MethodPost, productPath(p)+"/archive", token, nil)
		s.expect(res, http.StatusOK, "")
		res.decode(&p)
		if p.Status != "archived" {
			t.Fatalf("archive #%d: status = %q", i+1, p.Status)
		}
	}
	s.expect(s.do(http.MethodPut, productPath(p), token, renamed), http.StatusConflict, "product_archived")
	s.expect(s.do(http.MethodPatch, productPath(p)+"/stock", token, map[string]int{"stock": 1, "expected_stock": 50}), http.StatusConflict, "product_archived")
}

func TestResponsesAreJSON(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	for _, path := range []string{"/health", "/products", "/categories", "/shop", "/products/999999"} {
		res := s.do(http.MethodGet, path, sel.token, nil)
		if ct := res.header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("GET %s: Content-Type = %q, want JSON (status %d)", path, ct, res.status)
		}
	}
}
