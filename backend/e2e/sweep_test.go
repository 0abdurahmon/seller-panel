//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

func (s *server) writeAged(shopID int64, name string, age time.Duration) string {
	s.t.Helper()
	dir := filepath.Join(s.dir, fmt.Sprint(shopID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		s.t.Fatal(err)
	}
	at := time.Now().Add(-age)
	if err := os.Chtimes(path, at, at); err != nil {
		s.t.Fatal(err)
	}
	return path
}

func (s *server) age(url string, age time.Duration) {
	s.t.Helper()
	at := time.Now().Add(-age)
	if err := os.Chtimes(filepath.Join(s.dir, strings.TrimPrefix(url, "/api/uploads/")), at, at); err != nil {
		s.t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestSweepRemovesOnlyFilesNothingUses(t *testing.T) {
	s := newServer(t)
	a := s.register()
	cat := s.categoryID(a.token)
	png := pngFixture(t, 30, 30)

	var p product
	res := s.productForm("POST", "/products", a.token, createParts(cat, uniqueSKU(), photoPart(png), photoPart(png))...)
	s.expect(res, 201, "")
	res.decode(&p)
	var shop shopJSON
	res = s.putShop(a.token, shopParts("With logo", logoPart(png))...)
	s.expect(res, 200, "")
	res.decode(&shop)
	for _, u := range append(p.Photos, *shop.LogoURL) {
		s.age(u, 3*time.Hour)
	}

	orphan := s.writeAged(p.ShopID, strings.Repeat("e", 32)+".webp", 3*time.Hour)
	fresh := s.writeAged(p.ShopID, strings.Repeat("f", 32)+".webp", 5*time.Minute)
	temp := s.writeAged(p.ShopID, ".tmp-0123456789abcdef", 3*time.Hour)
	foreign := s.writeAged(p.ShopID, "notes.txt", 3*time.Hour)
	otherShopOrphan := s.writeAged(p.ShopID+1000, strings.Repeat("a", 32)+".webp", 3*time.Hour)

	got, err := s.app.SweepImages(t.Context(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got.Deleted != 3 {
		t.Fatalf("result = %+v, want 3 deleted", got)
	}
	for name, path := range map[string]string{"orphan": orphan, "temp file": temp, "orphan of another shop": otherShopOrphan} {
		if exists(path) {
			t.Errorf("the %s is still there", name)
		}
	}
	for name, path := range map[string]string{"new file": fresh, "foreign file": foreign} {
		if !exists(path) {
			t.Errorf("the %s was removed", name)
		}
	}
	for _, u := range append(p.Photos, *shop.LogoURL) {
		if !s.onDisk(u) {
			t.Errorf("%s is used and was removed", u)
		}
	}

	// A photo dropped by an update goes away with the update, not with the sweep.
	res = s.productForm("PUT", productPath(p), a.token, updateParts(cat, p.SKU, text("keep_photos", p.Photos[0]))...)
	s.expect(res, 200, "")
	if s.onDisk(p.Photos[1]) || !s.onDisk(p.Photos[0]) {
		t.Fatal("the update must delete the dropped photo and keep the other one")
	}
	if got, err := s.app.SweepImages(t.Context(), time.Hour); err != nil || got.Deleted != 0 {
		t.Fatalf("second sweep = %+v, %v; want nothing to delete", got, err)
	}
}

// Files are matched by /<shop_id>/<name>, so changing UPLOAD_PUBLIC_URL does not make a live image look unused.
func TestSweepKeepsImagesWhenThePublicURLChanged(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.UploadPublicURL = "https://cdn.example.com/media" })
	a := s.register()
	cat := s.categoryID(a.token)
	res := s.productForm("POST", "/products", a.token, createParts(cat, uniqueSKU(), photoPart(pngFixture(t, 20, 20)))...)
	s.expect(res, 201, "")
	var p product
	res.decode(&p)
	if !strings.HasPrefix(p.Photos[0], "https://cdn.example.com/media/") {
		t.Fatalf("photo url = %s", p.Photos[0])
	}
	rel := strings.TrimPrefix(p.Photos[0], "https://cdn.example.com/media/")
	at := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(filepath.Join(s.dir, rel), at, at); err != nil {
		t.Fatal(err)
	}

	got, err := s.app.SweepImages(t.Context(), time.Hour)
	if err != nil || got.Deleted != 0 {
		t.Fatalf("sweep = %+v, %v; want the used image kept", got, err)
	}
	if !exists(filepath.Join(s.dir, rel)) {
		t.Fatal("the image of a product was removed")
	}
}
