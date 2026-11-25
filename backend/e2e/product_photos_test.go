//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

type formPart struct {
	field string
	value string
	file  []byte
}

func text(field, value string) formPart { return formPart{field: field, value: value} }
func photoPart(data []byte) formPart    { return formPart{field: "photos", file: data} }

func (s *server) productForm(method, path, token string, parts ...formPart) response {
	s.t.Helper()
	return s.productFormTo(method, path, token, parts...)
}

func (s *server) productFormTo(method, path, token string, parts ...formPart) response {
	s.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range parts {
		if p.file != nil {
			fw, err := mw.CreateFormFile(p.field, "photo.png")
			if err != nil {
				s.t.Fatal(err)
			}
			if _, err := fw.Write(p.file); err != nil {
				s.t.Fatal(err)
			}
			continue
		}
		if err := mw.WriteField(p.field, p.value); err != nil {
			s.t.Fatal(err)
		}
	}
	mw.Close()
	req, err := http.NewRequest(method, s.url+path, &body)
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return response{t: s.t, status: res.StatusCode, header: res.Header, body: raw}
}

func createParts(category int64, sku string, extra ...formPart) []formPart {
	return append([]formPart{
		text("category_id", fmt.Sprint(category)), text("title", "Phone "+sku), text("description", "Good phone"),
		text("price", "1500000"), text("sku", sku), text("stock", "5"),
	}, extra...)
}

func updateParts(category int64, sku string, extra ...formPart) []formPart {
	return append([]formPart{
		text("category_id", fmt.Sprint(category)), text("title", "Phone "+sku), text("price", "1500000"), text("sku", sku),
	}, extra...)
}

func (s *server) filesOnDisk() int {
	s.t.Helper()
	n := 0
	err := filepath.WalkDir(s.dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return err
	})
	if err != nil {
		s.t.Fatal(err)
	}
	return n
}

func (s *server) onDisk(url string) bool {
	_, err := os.Stat(filepath.Join(s.dir, strings.TrimPrefix(url, "/api/uploads/")))
	return err == nil
}

func TestProductPhotosJourney(t *testing.T) {
	s := newServer(t)
	a := s.register()
	cat := s.categoryID(a.token)
	png := pngFixture(t, 40, 30)
	sku := uniqueSKU()

	res := s.productForm(http.MethodPost, "/products", a.token, createParts(cat, sku, photoPart(png), photoPart(pngFixture(t, 50, 50)))...)
	s.expect(res, http.StatusCreated, "")
	var created product
	res.decode(&created)
	if len(created.Photos) != 2 || created.Title != "Phone "+sku || created.Stock != 5 || created.Status != "draft" {
		t.Fatalf("created = %+v", created)
	}
	for _, u := range created.Photos {
		if !strings.HasPrefix(u, "/api/uploads/") || !bytes.HasPrefix(s.storedFile(u), []byte("RIFF")) {
			t.Fatalf("photo %s is not a stored webp", u)
		}
	}

	var got product
	s.do(http.MethodGet, productPath(created), a.token, nil).decode(&got)
	if fmt.Sprint(got.Photos) != fmt.Sprint(created.Photos) {
		t.Fatalf("get photos = %v, want %v", got.Photos, created.Photos)
	}
	var page productPage
	s.do(http.MethodGet, "/products", a.token, nil).decode(&page)
	if len(page.Items) != 1 || len(page.Items[0].Photos) != 2 {
		t.Fatalf("list = %+v", page)
	}

	// A JSON edit does not touch the photos.
	jsonUpdate := productBody(cat, sku, 0)
	delete(jsonUpdate, "stock")
	jsonUpdate["title"] = "Renamed"
	s.expect(s.do(http.MethodPut, productPath(created), a.token, jsonUpdate), http.StatusOK, "")
	s.do(http.MethodGet, productPath(created), a.token, nil).decode(&got)
	if got.Title != "Renamed" || fmt.Sprint(got.Photos) != fmt.Sprint(created.Photos) {
		t.Fatalf("after a JSON edit: %+v", got)
	}

	// Without keep_photos every photo stays and the new one is added.
	res = s.productForm(http.MethodPut, productPath(created), a.token, updateParts(cat, sku, photoPart(pngFixture(t, 20, 20)))...)
	s.expect(res, http.StatusOK, "")
	res.decode(&got)
	if len(got.Photos) != 3 || got.Photos[0] != created.Photos[0] || got.Photos[1] != created.Photos[1] {
		t.Fatalf("photos = %v", got.Photos)
	}

	// keep_photos picks the survivors in order: the dropped files are deleted from disk.
	first, second, third := got.Photos[0], got.Photos[1], got.Photos[2]
	res = s.productForm(http.MethodPut, productPath(created), a.token, updateParts(cat, sku, text("keep_photos", third), text("keep_photos", first))...)
	s.expect(res, http.StatusOK, "")
	res.decode(&got)
	if fmt.Sprint(got.Photos) != fmt.Sprint([]string{third, first}) {
		t.Fatalf("photos = %v, want [%s %s]", got.Photos, third, first)
	}
	if s.onDisk(second) || !s.onDisk(first) || !s.onDisk(third) {
		t.Fatalf("on disk: dropped %v, kept %v and %v; want only the dropped one gone", s.onDisk(second), s.onDisk(first), s.onDisk(third))
	}

	// The last photo cannot be removed.
	res = s.productForm(http.MethodPut, productPath(created), a.token, updateParts(cat, sku, text("keep_photos", ""))...)
	s.expect(res, http.StatusBadRequest, "validation_error")
	if s.filesOnDisk() != 2 {
		t.Fatalf("%d files on disk, want the two kept ones", s.filesOnDisk())
	}
}

func TestProductWithPhotosIsAllOrNothing(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	cat := s.categoryID(a.token)
	png := pngFixture(t, 30, 30)

	taken := s.createProduct(a.token, cat, 1)
	other := s.createProduct(a.token, cat, 1)
	before := s.filesOnDisk()

	// The save fails after the photos are converted: no file stays.
	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, taken.SKU, photoPart(png), photoPart(png))...), http.StatusConflict, "sku_taken")
	if n := s.filesOnDisk(); n != before {
		t.Fatalf("a duplicate SKU left %d files", n-before)
	}
	// The second photo is not an image: the first one does not stay either.
	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png), photoPart([]byte("not an image")))...), http.StatusUnsupportedMediaType, "unsupported_media_type")
	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png), photoPart(png[:40]))...), http.StatusBadRequest, "validation_error")
	// Bad fields and too many photos are refused before any image is processed.
	bad := createParts(cat, uniqueSKU(), photoPart(png))
	bad[1] = text("title", "")
	s.expect(s.productForm(http.MethodPost, "/products", a.token, bad...), http.StatusBadRequest, "validation_error")
	six := createParts(cat, uniqueSKU())
	for i := 0; i < 6; i++ {
		six = append(six, photoPart(png))
	}
	s.expect(s.productForm(http.MethodPost, "/products", a.token, six...), http.StatusBadRequest, "validation_error")
	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(bytes.Repeat([]byte{0x89}, 5<<20+1)))...), http.StatusRequestEntityTooLarge, "file_too_large")
	s.expect(s.productForm(http.MethodPost, "/products", "", createParts(cat, uniqueSKU(), photoPart(png))...), http.StatusUnauthorized, "unauthorized")
	if n := s.filesOnDisk(); n != before {
		t.Fatalf("refused requests left %d files", n-before)
	}
	var page productPage
	s.do(http.MethodGet, "/products", a.token, nil).decode(&page)
	if page.Total != 2 {
		t.Fatalf("%d products, want only the two created directly", page.Total)
	}

	// An update that fails keeps the old photos and removes the new file.
	res := s.productForm(http.MethodPut, productPath(other), a.token, updateParts(cat, other.SKU, photoPart(png))...)
	s.expect(res, http.StatusOK, "")
	var withPhoto product
	res.decode(&withPhoto)
	if s.filesOnDisk() != before+1 {
		t.Fatalf("%d files, want %d", s.filesOnDisk(), before+1)
	}
	s.expect(s.productForm(http.MethodPut, productPath(withPhoto), a.token, updateParts(cat, taken.SKU, text("keep_photos", ""), photoPart(png))...), http.StatusConflict, "sku_taken")
	var after product
	s.do(http.MethodGet, productPath(withPhoto), a.token, nil).decode(&after)
	if fmt.Sprint(after.Photos) != fmt.Sprint(withPhoto.Photos) || s.filesOnDisk() != before+1 || !s.onDisk(withPhoto.Photos[0]) {
		t.Fatalf("a failed update changed the product: photos %v, %d files", after.Photos, s.filesOnDisk())
	}

	// Another seller can neither change the product nor keep its photos.
	s.expect(s.productForm(http.MethodPut, productPath(withPhoto), b.token, updateParts(cat, withPhoto.SKU, photoPart(png))...), http.StatusNotFound, "not_found")
	mine := s.createProduct(b.token, s.categoryID(b.token), 1)
	s.expect(s.productForm(http.MethodPut, productPath(mine), b.token, updateParts(s.categoryID(b.token), mine.SKU, text("keep_photos", withPhoto.Photos[0]))...), http.StatusBadRequest, "validation_error")
	if s.filesOnDisk() != before+2 || !s.onDisk(withPhoto.Photos[0]) {
		t.Fatalf("another seller's requests changed the files: %d", s.filesOnDisk())
	}
}

func TestProductJSONRefusesAPhotosList(t *testing.T) {
	s := newServer(t)
	a := s.register()
	body := productBody(s.categoryID(a.token), uniqueSKU(), 1)
	body["photos"] = []string{"/api/uploads/1/" + strings.Repeat("a", 32) + ".webp"}
	s.expect(s.do(http.MethodPost, "/products", a.token, body), http.StatusBadRequest, "invalid_body")
}

func TestProductsWithPhotosShareTheUploadLimit(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.ImagesPerMinute = 2 })
	a, b := s.register(), s.register()
	cat := s.categoryID(a.token)
	png := pngFixture(t, 10, 10)

	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png))...), http.StatusCreated, "")
	s.expect(s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png))...), http.StatusCreated, "")
	limited := s.productForm(http.MethodPost, "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png))...)
	s.expect(limited, http.StatusTooManyRequests, "rate_limited")
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("Retry-After missing")
	}
	// A JSON request stores no image, so it is refused as invalid and never counted against the limit.
	s.expect(s.do(http.MethodPost, "/products", a.token, productBody(cat, uniqueSKU(), 1)), http.StatusBadRequest, "validation_error")
	s.expect(s.productForm(http.MethodPost, "/products", b.token, createParts(s.categoryID(b.token), uniqueSKU(), photoPart(png))...), http.StatusCreated, "")
}
