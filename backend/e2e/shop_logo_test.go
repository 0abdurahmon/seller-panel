//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

type shopJSON struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	LogoURL       *string `json:"logo_url"`
	Description   string  `json:"description"`
	ContactPhone  *string `json:"contact_phone"`
	PickupAddress *string `json:"pickup_address"`
}

func shopParts(name string, extra ...formPart) []formPart {
	return append([]formPart{text("name", name), text("description", "About"), text("contact_phone", "+998 90 123 45 67")}, extra...)
}

func logoPart(data []byte) formPart { return formPart{field: "logo", file: data} }

func (s *server) putShop(token string, parts ...formPart) response {
	s.t.Helper()
	return s.productFormTo(http.MethodPut, "/shop", token, parts...)
}

func TestShopLogoJourney(t *testing.T) {
	s := newServer(t)
	a := s.register()
	png := pngFixture(t, 64, 64)

	res := s.putShop(a.token, shopParts("Logo shop", logoPart(png))...)
	s.expect(res, http.StatusOK, "")
	var shop shopJSON
	res.decode(&shop)
	if shop.Name != "Logo shop" || shop.LogoURL == nil || !strings.HasPrefix(*shop.LogoURL, fmt.Sprintf("/api/uploads/%d/", shop.ID)) {
		t.Fatalf("shop = %+v", shop)
	}
	first := *shop.LogoURL
	if !strings.HasPrefix(string(s.storedFile(first)), "RIFF") {
		t.Fatal("the logo is not a stored webp")
	}
	var got shopJSON
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&got)
	if got.LogoURL == nil || *got.LogoURL != first || got.ContactPhone == nil || *got.ContactPhone != "+998901234567" {
		t.Fatalf("GET /shop = %+v", got)
	}

	// A JSON edit keeps the logo.
	s.expect(s.do(http.MethodPut, "/shop", a.token, map[string]string{"name": "Renamed", "description": "x"}), http.StatusOK, "")
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&got)
	if got.Name != "Renamed" || got.LogoURL == nil || *got.LogoURL != first || !s.onDisk(first) {
		t.Fatalf("after a JSON edit: %+v, logo on disk %v", got, s.onDisk(first))
	}

	// A new logo replaces the old one: the old file is deleted.
	res = s.putShop(a.token, shopParts("Renamed", logoPart(pngFixture(t, 32, 32)))...)
	s.expect(res, http.StatusOK, "")
	res.decode(&got)
	if got.LogoURL == nil || *got.LogoURL == first || s.onDisk(first) || !s.onDisk(*got.LogoURL) || s.filesOnDisk() != 1 {
		t.Fatalf("replace: logo %v, old file on disk %v, %d files", got.LogoURL, s.onDisk(first), s.filesOnDisk())
	}

	// A multipart edit without a logo keeps it; remove_logo clears it and deletes the file.
	s.expect(s.putShop(a.token, shopParts("Renamed")...), http.StatusOK, "")
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&got)
	if got.LogoURL == nil || s.filesOnDisk() != 1 {
		t.Fatalf("a multipart edit without a logo lost it: %+v", got)
	}
	res = s.putShop(a.token, shopParts("Renamed", text("remove_logo", "true"))...)
	s.expect(res, http.StatusOK, "")
	if !strings.Contains(string(res.body), `"logo_url":null`) || s.filesOnDisk() != 0 {
		t.Fatalf("remove: %s, %d files on disk", res.body, s.filesOnDisk())
	}
}

func TestShopUpdateWithALogoIsAllOrNothing(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	png := pngFixture(t, 30, 30)

	s.expect(s.putShop(a.token, shopParts("Has logo", logoPart(png))...), http.StatusOK, "")
	if s.filesOnDisk() != 1 {
		t.Fatalf("%d files", s.filesOnDisk())
	}
	var before shopJSON
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&before)

	bad := map[string][]formPart{
		"not an image":     shopParts("Changed", logoPart([]byte("not an image"))),
		"a broken image":   shopParts("Changed", logoPart(png[:40])),
		"an empty name":    {text("name", ""), logoPart(png)},
		"a bad phone":      {text("name", "Changed"), text("contact_phone", "abc"), logoPart(png)},
		"logo and remove":  shopParts("Changed", logoPart(png), text("remove_logo", "true")),
		"a logo_url field": shopParts("Changed", text("logo_url", "https://evil.example.com/a.png")),
	}
	want := map[string]int{
		"not an image": http.StatusUnsupportedMediaType, "a broken image": http.StatusBadRequest, "an empty name": http.StatusBadRequest,
		"a bad phone": http.StatusBadRequest, "logo and remove": http.StatusBadRequest, "a logo_url field": http.StatusBadRequest,
	}
	for name, parts := range bad {
		if res := s.putShop(a.token, parts...); res.status != want[name] {
			t.Errorf("%s: status %d, want %d: %s", name, res.status, want[name], res.body)
		}
	}
	var after shopJSON
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&after)
	if after.Name != before.Name || after.LogoURL == nil || *after.LogoURL != *before.LogoURL || s.filesOnDisk() != 1 || !s.onDisk(*before.LogoURL) {
		t.Fatalf("refused requests changed the shop: %+v, %d files", after, s.filesOnDisk())
	}

	s.expect(s.putShop(a.token, shopParts("Big", logoPart(append([]byte{0x89}, make([]byte, 5<<20)...)))...), http.StatusRequestEntityTooLarge, "file_too_large")
	s.expect(s.putShop("", shopParts("Anon", logoPart(png))...), http.StatusUnauthorized, "unauthorized")
	s.expect(s.do(http.MethodPut, "/shop", a.token, map[string]string{"name": "X", "logo_url": "https://evil.example.com/a.png"}), http.StatusBadRequest, "invalid_body")

	// Another seller's logo goes to their own shop and never touches the first one.
	res := s.putShop(b.token, shopParts("Other", logoPart(png))...)
	s.expect(res, http.StatusOK, "")
	var other shopJSON
	res.decode(&other)
	if other.ID == before.ID || other.LogoURL == nil || *other.LogoURL == *before.LogoURL {
		t.Fatalf("seller B got %+v, seller A has %+v", other, before)
	}
	s.do(http.MethodGet, "/shop", a.token, nil).decode(&after)
	if after.LogoURL == nil || *after.LogoURL != *before.LogoURL || s.filesOnDisk() != 2 {
		t.Fatalf("seller B's upload changed seller A's shop: %+v, %d files", after, s.filesOnDisk())
	}
}

func TestShopLogoSharesTheUploadLimit(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.UploadsPerMinute = 2 })
	a := s.register()
	png := pngFixture(t, 10, 10)

	s.expect(s.putShop(a.token, shopParts("One", logoPart(png))...), http.StatusOK, "")
	s.expect(s.putShop(a.token, shopParts("Two", logoPart(png))...), http.StatusOK, "")
	limited := s.putShop(a.token, shopParts("Three", logoPart(png))...)
	s.expect(limited, http.StatusTooManyRequests, "rate_limited")
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("Retry-After missing")
	}
	s.expect(s.do(http.MethodPut, "/shop", a.token, map[string]string{"name": "JSON edits are not counted"}), http.StatusOK, "")
}
