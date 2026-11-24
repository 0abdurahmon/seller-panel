//go:build e2e

package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func (s *server) storedFile(url string) []byte {
	s.t.Helper()
	data, err := os.ReadFile(filepath.Join(s.dir, strings.TrimPrefix(url, "/api/uploads/")))
	if err != nil {
		s.t.Fatalf("the uploaded file is not on disk: %v", err)
	}
	return data
}

// Images are saved by POST /products, PUT /products/{id} and PUT /shop; there is no endpoint of their own.
func TestThereIsNoUploadEndpoint(t *testing.T) {
	s := newServer(t)
	a := s.register()
	s.expect(s.productFormTo(http.MethodPost, "/uploads", a.token, photoPart(pngFixture(t, 10, 10))), http.StatusNotFound, "not_found")
	if n := s.filesOnDisk(); n != 0 {
		t.Fatalf("%d files stored by a route that does not exist", n)
	}
}
