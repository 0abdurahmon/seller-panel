//go:build e2e

package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

func pngFixture(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func (s *server) upload(token, field, filename string, data []byte) response {
	s.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		s.t.Fatal(err)
	}
	fw.Write(data)
	mw.Close()
	req, err := http.NewRequest(http.MethodPost, s.url+"/uploads", &body)
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

func (s *server) storedFile(url string) []byte {
	s.t.Helper()
	data, err := os.ReadFile(filepath.Join(s.dir, strings.TrimPrefix(url, "/api/uploads/")))
	if err != nil {
		s.t.Fatalf("the uploaded file is not on disk: %v", err)
	}
	return data
}

type uploaded struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func TestUploadJourney(t *testing.T) {
	s := newServer(t)
	a := s.register()

	res := s.upload(a.token, "file", "photo.png", pngFixture(t, 200, 100))
	s.expect(res, http.StatusCreated, "")
	var img uploaded
	res.decode(&img)
	if img.ContentType != "image/webp" || img.Width != 200 || img.Height != 100 || img.Size <= 0 || !strings.HasPrefix(img.URL, "/api/uploads/") {
		t.Fatalf("unexpected upload %+v", img)
	}

	stored := s.storedFile(img.URL)
	if !bytes.HasPrefix(stored, []byte("RIFF")) || len(stored) != img.Size {
		t.Fatalf("stored %d bytes, want a webp of %d bytes", len(stored), img.Size)
	}
	s.expect(s.do(http.MethodGet, strings.TrimPrefix(img.URL, "/api"), "", nil), http.StatusNotFound, "not_found")

	var logo = map[string]string{"name": "Shop", "logo_url": "https://example.com" + img.URL}
	s.expect(s.do(http.MethodPut, "/shop", a.token, logo), http.StatusOK, "")
}

func TestUploadErrors(t *testing.T) {
	s := newServer(t)
	a := s.register()
	png := pngFixture(t, 10, 10)

	s.expect(s.upload("", "file", "a.png", png), http.StatusUnauthorized, "unauthorized")
	s.expect(s.upload("bad.token.value", "file", "a.png", png), http.StatusUnauthorized, "unauthorized")
	s.expect(s.upload(a.token, "avatar", "a.png", png), http.StatusBadRequest, "validation_error")
	s.expect(s.upload(a.token, "file", "a.png", nil), http.StatusBadRequest, "validation_error")
	s.expect(s.upload(a.token, "file", "a.png", []byte("not an image")), http.StatusUnsupportedMediaType, "unsupported_media_type")
	s.expect(s.upload(a.token, "file", "a.png", png[:40]), http.StatusBadRequest, "validation_error")
	s.expect(s.upload(a.token, "file", "a.png", bytes.Repeat([]byte{0x89}, 5<<20+1)), http.StatusRequestEntityTooLarge, "file_too_large")
	s.expect(s.do(http.MethodPost, "/uploads", a.token, map[string]string{"file": "x"}), http.StatusUnsupportedMediaType, "unsupported_media_type")
	s.expect(s.do(http.MethodGet, "/uploads", a.token, nil), http.StatusMethodNotAllowed, "method_not_allowed")
}

func TestUploadedFilesLandInTheirOwnShopDirectory(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()

	var imgA, imgB uploaded
	s.upload(a.token, "file", "a.png", pngFixture(t, 20, 20)).decode(&imgA)
	s.upload(b.token, "file", "b.png", pngFixture(t, 20, 20)).decode(&imgB)

	rel := func(u uploaded) string { return strings.TrimPrefix(u.URL, "/api/uploads/") }
	if shopA, shopB := strings.Split(rel(imgA), "/")[0], strings.Split(rel(imgB), "/")[0]; shopA == shopB {
		t.Fatalf("both sellers share the directory %s", shopA)
	}
	name := rel(imgA)[strings.Index(rel(imgA), "/")+1:]
	shopB := strings.Split(rel(imgB), "/")[0]
	if _, err := os.Stat(filepath.Join(s.dir, shopB, name)); err == nil {
		t.Fatal("seller A's file is in seller B's directory")
	}
}

func TestUploadsAreLimitedPerShop(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.UploadsPerMinute = 2 })
	a, b := s.register(), s.register()
	data := pngFixture(t, 10, 10)

	s.expect(s.upload(a.token, "file", "a.png", data), http.StatusCreated, "")
	s.expect(s.upload(a.token, "file", "a.png", data), http.StatusCreated, "")
	limited := s.upload(a.token, "file", "a.png", data)
	s.expect(limited, http.StatusTooManyRequests, "rate_limited")
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("Retry-After missing")
	}
	s.expect(s.upload(b.token, "file", "b.png", data), http.StatusCreated, "")
	s.expect(s.do(http.MethodGet, "/products", a.token, nil), http.StatusOK, "")
}
