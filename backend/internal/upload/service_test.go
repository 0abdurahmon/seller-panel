package upload

import (
	"bytes"
	"context"
	"errors"
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gen2brain/webp"
)

func decodeStored(t *testing.T, repo *memRepository, shopID int64, url string) image.Image {
	t.Helper()
	name := url[strings.LastIndex(url, "/")+1:]
	f, err := repo.Open(context.Background(), shopID, name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := webp.Decode(f)
	if err != nil {
		t.Fatalf("stored file is not webp: %v", err)
	}
	return img
}

func TestUploadConvertsToWebP(t *testing.T) {
	huge := bytes.Repeat([]byte{0}, MaxFileBytes+1)
	tests := []struct {
		name    string
		data    []byte
		wantErr error
		wantW   int
		wantH   int
	}{
		{"jpeg", jpegBytes(t, 300, 200), nil, 300, 200},
		{"png", pngBytes(t, 120, 80), nil, 120, 80},
		{"webp", webpBytes(t, 64, 64), nil, 64, 64},
		{"wide image is scaled down", pngBytes(t, 4096, 1024), nil, 2048, 512},
		{"tall image is scaled down", pngBytes(t, 1000, 3000), nil, 682, 2048},
		{"exif rotate 90 swaps sides", withOrientation(jpegBytes(t, 300, 200), 6), nil, 200, 300},
		{"empty", nil, ErrValidation, 0, 0},
		{"gif is rejected", gifBytes(t), ErrUnsupportedType, 0, 0},
		{"text renamed to image", []byte("<?php echo 1; ?>"), ErrUnsupportedType, 0, 0},
		{"svg is rejected", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ErrUnsupportedType, 0, 0},
		{"truncated jpeg", jpegBytes(t, 300, 200)[:200], ErrValidation, 0, 0},
		{"jpeg header with garbage", append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{7}, 50)...), ErrValidation, 0, 0},
		{"too many pixels", pngBytes(t, 6000, 4000), ErrValidation, 0, 0},
		{"over 5 MB", huge, ErrTooLarge, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemRepository()
			svc := NewService(repo, "/api/uploads/")
			got, err := svc.Upload(context.Background(), 7, bytes.NewReader(tc.data))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if repo.count() != 0 {
					t.Fatalf("a rejected upload stored %d files", repo.count())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Width != tc.wantW || got.Height != tc.wantH {
				t.Fatalf("size = %dx%d, want %dx%d", got.Width, got.Height, tc.wantW, tc.wantH)
			}
			if !strings.HasPrefix(got.URL, "/api/uploads/7/") || !strings.HasSuffix(got.URL, ".webp") || got.ContentType != ContentType {
				t.Fatalf("unexpected result %+v", got)
			}
			img := decodeStored(t, repo, 7, got.URL)
			if b := img.Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
				t.Fatalf("stored size = %dx%d", b.Dx(), b.Dy())
			}
		})
	}
}

func TestUploadStripsMetadata(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	got, err := svc.Upload(context.Background(), 1, bytes.NewReader(withOrientation(jpegBytes(t, 50, 50), 3)))
	if err != nil {
		t.Fatal(err)
	}
	name := got.URL[strings.LastIndex(got.URL, "/")+1:]
	data := repo.files[key(1, name)]
	if bytes.Contains(data, []byte("EXIF")) || bytes.Contains(data, []byte("Exif")) {
		t.Fatal("stored file still carries EXIF")
	}
}

func TestUploadNamesAreUnique(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	data := pngBytes(t, 10, 10)
	a, _ := svc.Upload(context.Background(), 1, bytes.NewReader(data))
	b, _ := svc.Upload(context.Background(), 1, bytes.NewReader(data))
	if a.URL == b.URL || repo.count() != 2 {
		t.Fatalf("same image twice must give two files: %q %q", a.URL, b.URL)
	}
}

func TestOpenOtherShopsFileIsNotFound(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	got, err := svc.Upload(context.Background(), 1, bytes.NewReader(pngBytes(t, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}
	name := got.URL[strings.LastIndex(got.URL, "/")+1:]
	if _, err := svc.Open(context.Background(), 2, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v, want ErrNotFound", err)
	}
	f, err := svc.Open(context.Background(), 1, name)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestOpenRejectsForeignNames(t *testing.T) {
	svc := NewService(newMemRepository(), "/u")
	for _, name := range []string{"", "../1/a.webp", "..%2f", "a.webp", strings.Repeat("a", 32) + ".png", strings.Repeat("A", 32) + ".webp", ".tmp-0011223344556677"} {
		if _, err := svc.Open(context.Background(), 1, name); !errors.Is(err, ErrNotFound) {
			t.Errorf("name %q: err = %v, want ErrNotFound", name, err)
		}
	}
	if _, err := svc.Open(context.Background(), 0, strings.Repeat("a", 32)+".webp"); !errors.Is(err, ErrNotFound) {
		t.Error("shop 0 must be not found")
	}
}

func TestUploadIsBusyWhenAllSlotsAreTaken(t *testing.T) {
	svc := NewService(newMemRepository(), "/u")
	for i := 0; i < maxConcurrent; i++ {
		svc.slots <- struct{}{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), busyWait+2*time.Second)
	defer cancel()
	_, err := svc.Upload(ctx, 1, bytes.NewReader(pngBytes(t, 10, 10)))
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
}

func TestUploadConcurrently(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	data := jpegBytes(t, 200, 200)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Upload(context.Background(), 1, bytes.NewReader(data)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if repo.count() != 12 {
		t.Fatalf("stored %d files, want 12", repo.count())
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	got, err := svc.Upload(ctx, 1, bytes.NewReader(pngBytes(t, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}
	name := got.URL[strings.LastIndex(got.URL, "/")+1:]

	tests := []struct {
		name    string
		shop    int64
		file    string
		wantErr error
	}{
		{"other shop gets not found", 2, name, ErrNotFound},
		{"foreign name", 1, "../1/" + name, ErrNotFound},
		{"unknown file", 1, strings.Repeat("0", 32) + ".webp", ErrNotFound},
		{"shop zero", 0, name, ErrNotFound},
		{"owner deletes", 1, name, nil},
		{"second delete is not found", 1, name, ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Delete(ctx, tc.shop, tc.file); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
	if repo.count() != 0 {
		t.Fatalf("%d files left", repo.count())
	}
}

func TestRemove(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	svc := NewService(repo, "/api/uploads")
	got, err := svc.Upload(ctx, 1, bytes.NewReader(pngBytes(t, 10, 10)))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		shop    int64
		url     string
		wantErr error
	}{
		{"another shop", 2, got.URL, ErrNotFound},
		{"external url", 1, "https://example.com/api/uploads/1/" + strings.Repeat("a", 32) + ".webp", ErrNotFound},
		{"other prefix", 1, strings.Replace(got.URL, "/api/uploads", "/files", 1), ErrNotFound},
		{"foreign name", 1, "/api/uploads/1/../1/x.webp", ErrNotFound},
		{"unknown file", 1, "/api/uploads/1/" + strings.Repeat("0", 32) + ".webp", ErrNotFound},
		{"not a url", 1, "", ErrNotFound},
		{"owner removes it", 1, got.URL, nil},
		{"second remove", 1, got.URL, ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Remove(ctx, tc.shop, tc.url); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
	if repo.count() != 0 {
		t.Fatalf("%d files left", repo.count())
	}
}
