//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestSellerCannotTouchAnotherSellersProduct(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	category := s.categoryID(a.token)
	p := s.createProduct(a.token, category, 10)

	update := productBody(category, p.SKU, 0)
	delete(update, "stock")
	update["title"] = "Hacked"

	s.expect(s.do(http.MethodGet, productPath(p), b.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodPut, productPath(p), b.token, update), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodPost, productPath(p)+"/archive", b.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodPatch, productPath(p)+"/stock", b.token, map[string]int{"stock": 0, "expected_stock": 10}), http.StatusNotFound, "not_found")

	var page productPage
	s.do(http.MethodGet, "/products", b.token, nil).decode(&page)
	if page.Total != 0 || len(page.Items) != 0 || page.Items == nil {
		t.Fatalf("seller B list = %+v, want an empty page", page)
	}
	s.do(http.MethodGet, "/products?q="+strings.ToLower(p.SKU), b.token, nil).decode(&page)
	if page.Total != 0 {
		t.Fatalf("seller B finds seller A's product by SKU: %+v", page)
	}

	var mine product
	s.do(http.MethodGet, productPath(p), a.token, nil).decode(&mine)
	if mine.Title != p.Title || mine.Status != "active" || mine.Stock != 10 {
		t.Fatalf("seller A's product changed after B's attempts: %+v", mine)
	}
}

func TestShopIDCannotBeChosenByTheClient(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	category := s.categoryID(a.token)
	pa := s.createProduct(a.token, category, 1)

	body := productBody(category, uniqueSKU(), 1)
	body["shop_id"] = pa.ShopID
	s.expect(s.do(http.MethodPost, "/products", b.token, body), http.StatusBadRequest, "invalid_body")

	pb := s.createProduct(b.token, category, 1)
	if pb.ShopID == pa.ShopID {
		t.Fatalf("both sellers got shop %d", pb.ShopID)
	}
}

func TestEachSellerSeesOnlyTheirOwnShop(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()

	ra := s.do(http.MethodGet, "/shop", a.token, nil)
	rb := s.do(http.MethodGet, "/shop", b.token, nil)
	if !strings.Contains(string(ra.body), a.shopName) || strings.Contains(string(ra.body), b.shopName) {
		t.Fatalf("seller A sees %s", ra.body)
	}
	if !strings.Contains(string(rb.body), b.shopName) || strings.Contains(string(rb.body), a.shopName) {
		t.Fatalf("seller B sees %s", rb.body)
	}

	s.expect(s.do(http.MethodPut, "/shop", b.token, map[string]string{
		"name": "B changed it", "description": "", "logo_url": "", "contact_phone": "", "pickup_address": "",
	}), http.StatusOK, "")
	if again := s.do(http.MethodGet, "/shop", a.token, nil); strings.Contains(string(again.body), "B changed it") {
		t.Fatalf("seller B's update leaked into seller A's shop: %s", again.body)
	}
}

func TestSameSKUIsAllowedInDifferentShops(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	category := s.categoryID(a.token)
	sku := uniqueSKU()

	s.expect(s.do(http.MethodPost, "/products", a.token, productBody(category, sku, 1)), http.StatusCreated, "")
	s.expect(s.do(http.MethodPost, "/products", b.token, productBody(category, sku, 1)), http.StatusCreated, "")
	s.expect(s.do(http.MethodPost, "/products", a.token, productBody(category, sku, 1)), http.StatusConflict, "sku_taken")
}
