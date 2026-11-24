//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (s *server) uploadPNG(token string) string {
	s.t.Helper()
	res := s.upload(token, "file", "p.png", pngFixture(s.t, 40, 30))
	s.expect(res, http.StatusCreated, "")
	var img uploaded
	res.decode(&img)
	return img.URL
}

func TestProductPhotosJourney(t *testing.T) {
	s := newServer(t)
	a := s.register()
	cat := s.categoryID(a.token)
	p1, p2, p3 := s.uploadPNG(a.token), s.uploadPNG(a.token), s.uploadPNG(a.token)

	body := productBody(cat, uniqueSKU(), 5)
	body["photos"] = []string{p2, p1}
	res := s.do(http.MethodPost, "/products", a.token, body)
	s.expect(res, http.StatusCreated, "")
	var created product
	res.decode(&created)
	if fmt.Sprint(created.Photos) != fmt.Sprint([]string{p2, p1}) {
		t.Fatalf("photos = %v, want the cover first: %v", created.Photos, []string{p2, p1})
	}

	var got product
	s.do(http.MethodGet, productPath(created), a.token, nil).decode(&got)
	if fmt.Sprint(got.Photos) != fmt.Sprint(created.Photos) {
		t.Fatalf("get photos = %v", got.Photos)
	}
	if len(s.storedFile(got.Photos[0])) == 0 {
		t.Fatal("the cover photo is empty on disk")
	}

	var page productPage
	s.do(http.MethodGet, "/products", a.token, nil).decode(&page)
	if len(page.Items) != 1 || len(page.Items[0].Photos) != 2 {
		t.Fatalf("list = %+v", page)
	}

	update := productBody(cat, created.SKU, 0)
	delete(update, "stock")
	s.expect(s.do(http.MethodPut, productPath(created), a.token, update), http.StatusOK, "")
	s.do(http.MethodGet, productPath(created), a.token, nil).decode(&got)
	if len(got.Photos) != 2 {
		t.Fatalf("a PUT without photos must keep them, got %v", got.Photos)
	}

	update["photos"] = []string{p3}
	res = s.do(http.MethodPut, productPath(created), a.token, update)
	s.expect(res, http.StatusOK, "")
	res.decode(&got)
	if fmt.Sprint(got.Photos) != fmt.Sprint([]string{p3}) {
		t.Fatalf("replaced photos = %v", got.Photos)
	}

	update["photos"] = []string{}
	res = s.do(http.MethodPut, productPath(created), a.token, update)
	s.expect(res, http.StatusOK, "")
	if !strings.Contains(string(res.body), `"photos":[]`) {
		t.Fatalf("cleared photos must be [], got %s", res.body)
	}
}

func TestProductPhotosRejectBadURLs(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	cat := s.categoryID(a.token)
	mine, theirs := s.uploadPNG(a.token), s.uploadPNG(b.token)
	missing := strings.Replace(mine, mine[len(mine)-37:], strings.Repeat("0", 32)+".webp", 1)

	bad := map[string][]string{
		"another seller's upload": {theirs},
		"external url":            {"https://evil.example.com/a.webp"},
		"script url":              {"javascript:alert(1)"},
		"missing file":            {missing},
		"duplicate":               {mine, mine},
		"six photos":              {mine, mine + "?1", mine + "?2", mine + "?3", mine + "?4", mine + "?5"},
		"path trick":              {strings.Replace(mine, "/uploads/", "/uploads/../uploads/", 1)},
	}
	for name, photos := range bad {
		body := productBody(cat, uniqueSKU(), 1)
		body["photos"] = photos
		s.expect(s.do(http.MethodPost, "/products", a.token, body), http.StatusBadRequest, "validation_error")
		_ = name
	}

	var page productPage
	s.do(http.MethodGet, "/products", a.token, nil).decode(&page)
	if page.Total != 0 {
		t.Fatalf("%d products created by rejected requests", page.Total)
	}

	p := s.createProduct(a.token, cat, 1)
	update := productBody(cat, p.SKU, 0)
	delete(update, "stock")
	update["photos"] = []string{theirs}
	s.expect(s.do(http.MethodPut, productPath(p), a.token, update), http.StatusBadRequest, "validation_error")
	var got product
	s.do(http.MethodGet, productPath(p), a.token, nil).decode(&got)
	if len(got.Photos) != 0 {
		t.Fatalf("a rejected update changed the photos: %v", got.Photos)
	}
}
