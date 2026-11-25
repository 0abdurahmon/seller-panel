package product

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func img(n int) []byte { return []byte(fmt.Sprintf("image-%d", n)) }

func TestCreateStoresPhotosInOrder(t *testing.T) {
	store := newFakePhotos()
	svc := NewService(newMemRepository(), store)
	p, err := svc.Create(context.Background(), 1, validInput(), [][]byte{img(1), img(2), img(3)})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Photos) != 3 || p.Photos[0] != "/api/uploads/1/00000000000000000000000000000001.webp" {
		t.Fatalf("photos = %v", p.Photos)
	}
	if len(store.live()) != 3 {
		t.Fatalf("live files = %v", store.live())
	}
}

func TestCreateLeavesNoFileWhenTheProductIsRefused(t *testing.T) {
	ctx := context.Background()
	store := newFakePhotos()
	svc := NewService(newMemRepository(), store)
	if _, err := svc.Create(ctx, 1, validInput(), onePhoto()); err != nil {
		t.Fatal(err)
	}

	t.Run("duplicate sku", func(t *testing.T) {
		if _, err := svc.Create(ctx, 1, validInput(), [][]byte{img(1), img(2)}); !errors.Is(err, ErrSKUTaken) {
			t.Fatalf("err = %v, want ErrSKUTaken", err)
		}
		if live := store.live(); len(live) != 1 {
			t.Fatalf("files left behind: %v", live)
		}
	})
	t.Run("invalid fields never reach the photo store", func(t *testing.T) {
		before := store.next
		in := validInput()
		in.Title = ""
		if _, err := svc.Create(ctx, 1, in, [][]byte{img(1)}); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v", err)
		}
		if store.next != before {
			t.Fatal("an image was processed for an invalid product")
		}
	})
	t.Run("six photos", func(t *testing.T) {
		before := store.next
		in := validInput()
		in.SKU = "SIX"
		files := [][]byte{img(1), img(2), img(3), img(4), img(5), img(6)}
		if _, err := svc.Create(ctx, 1, in, files); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v", err)
		}
		if store.next != before {
			t.Fatal("images were processed for a product with too many photos")
		}
	})
	t.Run("third photo is refused", func(t *testing.T) {
		store.failAt = store.next + 3
		store.failErr = ErrUnsupportedMedia
		in := validInput()
		in.SKU = "THIRD"
		if _, err := svc.Create(ctx, 1, in, [][]byte{img(1), img(2), img(3)}); !errors.Is(err, ErrUnsupportedMedia) {
			t.Fatalf("err = %v, want ErrUnsupportedMedia", err)
		}
		if live := store.live(); len(live) != 1 {
			t.Fatalf("the first two photos stayed on disk: %v", live)
		}
		store.failAt = 0
	})
}

func TestFiveVersusSixPhotos(t *testing.T) {
	svc := newService()
	in := validInput()
	if p, err := svc.Create(context.Background(), 1, in, [][]byte{img(1), img(2), img(3), img(4), img(5)}); err != nil || len(p.Photos) != 5 {
		t.Fatalf("five photos: %v, %v", p.Photos, err)
	}
}

func TestUpdatePhotos(t *testing.T) {
	ctx := context.Background()
	store := newFakePhotos()
	svc := NewService(newMemRepository(), store)
	p, err := svc.Create(ctx, 1, validInput(), [][]byte{img(1), img(2), img(3)})
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := p.Photos[0], p.Photos[1], p.Photos[2]
	up := UpdateInput{CategoryID: 1, Title: "New", Price: 5, SKU: "PX-1"}
	keep := func(urls ...string) *[]string { return &urls }

	t.Run("no photo field keeps all", func(t *testing.T) {
		got, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{})
		if err != nil || fmt.Sprint(got.Photos) != fmt.Sprint([]string{a, b, c}) || len(store.removed) != 0 {
			t.Fatalf("photos = %v, removed = %v, err = %v", got.Photos, store.removed, err)
		}
	})
	t.Run("keep a list and add one", func(t *testing.T) {
		got, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Keep: keep(c, a), Add: [][]byte{img(9)}})
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Photos) != 3 || got.Photos[0] != c || got.Photos[1] != a {
			t.Fatalf("photos = %v, want c, a, new", got.Photos)
		}
		if fmt.Sprint(store.removed) != fmt.Sprint([]string{b}) {
			t.Fatalf("removed = %v, want only b", store.removed)
		}
		p.Photos = got.Photos
		a, b, c = got.Photos[0], got.Photos[1], got.Photos[2]
	})
	t.Run("keep nothing and add one replaces all", func(t *testing.T) {
		store.removed = nil
		got, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Keep: keep(), Add: [][]byte{img(10)}})
		if err != nil || len(got.Photos) != 1 || len(store.removed) != 3 {
			t.Fatalf("photos = %v, removed = %v, err = %v", got.Photos, store.removed, err)
		}
	})
}

func TestUpdatePhotoErrors(t *testing.T) {
	ctx := context.Background()
	store := newFakePhotos()
	svc := NewService(newMemRepository(), store)
	p, _ := svc.Create(ctx, 1, validInput(), [][]byte{img(1), img(2)})
	other := validInput()
	other.SKU = "OTHER"
	q, _ := svc.Create(ctx, 1, other, [][]byte{img(3)})
	up := UpdateInput{CategoryID: 1, Title: "New", Price: 5, SKU: "PX-1"}
	list := func(urls ...string) *[]string { return &urls }
	savedBefore := len(store.live())

	cases := map[string]PhotoChange{
		"keep a photo of another product": {Keep: list(q.Photos[0])},
		"keep an unknown url":             {Keep: list("/api/uploads/1/" + strings.Repeat("f", 32) + ".webp")},
		"keep the same photo twice":       {Keep: list(p.Photos[0], p.Photos[0])},
		"too many in total":               {Add: [][]byte{img(5), img(6), img(7), img(8)}},
	}
	for name, pc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Update(ctx, 1, p.ID, up, pc); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
			if len(store.live()) != savedBefore || len(store.removed) != 0 {
				t.Fatalf("files changed by a refused update: live %v, removed %v", store.live(), store.removed)
			}
		})
	}

	t.Run("another shop's product", func(t *testing.T) {
		if _, err := svc.Update(ctx, 2, p.ID, up, PhotoChange{Add: [][]byte{img(1)}}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
		if len(store.live()) != savedBefore {
			t.Fatal("a file was stored for another shop's product")
		}
	})
	t.Run("sku conflict removes the new file and keeps the old ones", func(t *testing.T) {
		clash := up
		clash.SKU = "OTHER"
		if _, err := svc.Update(ctx, 1, p.ID, clash, PhotoChange{Keep: list(), Add: [][]byte{img(4)}}); !errors.Is(err, ErrSKUTaken) {
			t.Fatalf("err = %v, want ErrSKUTaken", err)
		}
		if len(store.live()) != savedBefore {
			t.Fatalf("live = %v, the new file must be removed and the old ones kept", store.live())
		}
		got, _ := svc.Get(ctx, 1, p.ID)
		if len(got.Photos) != 2 {
			t.Fatalf("photos = %v", got.Photos)
		}
	})
	t.Run("a failing second photo removes the first new one", func(t *testing.T) {
		store.failAt = store.next + 2
		store.failErr = ErrBusy
		_, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(5), img(6)}})
		store.failAt = 0
		if !errors.Is(err, ErrBusy) {
			t.Fatalf("err = %v, want ErrBusy", err)
		}
		if len(store.live()) != savedBefore {
			t.Fatalf("live = %v", store.live())
		}
	})
}

func TestArchiveKeepsPhotos(t *testing.T) {
	ctx := context.Background()
	store := newFakePhotos()
	svc := NewService(newMemRepository(), store)
	p, _ := svc.Create(ctx, 1, validInput(), [][]byte{img(1)})
	got, err := svc.Archive(ctx, 1, p.ID)
	if err != nil || len(got.Photos) != 1 || len(store.removed) != 0 {
		t.Fatalf("photos = %v, removed = %v, err = %v", got.Photos, store.removed, err)
	}
}

// --- HTTP ---

type formFile struct {
	field string
	name  string
	data  []byte
}

func multipartBody(t *testing.T, fields [][2]string, files []formFile) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, f := range fields {
		if err := mw.WriteField(f[0], f[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		fw, err := mw.CreateFormFile(f.field, f.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(f.data); err != nil {
			t.Fatal(err)
		}
	}
	mw.Close()
	return &body, mw.FormDataContentType()
}

func (a *api) multipart(method, path, token string, fields [][2]string, files []formFile) *httptest.ResponseRecorder {
	a.t.Helper()
	body, ct := multipartBody(a.t, fields, files)
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", ct)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec
}

var baseFields = func() [][2]string {
	return [][2]string{{"category_id", "1"}, {"title", "Phone X"}, {"description", "Good"}, {"price", "1000000"}, {"sku", "PX-1"}, {"stock", "10"}}
}

func photo(n int) formFile { return formFile{"photos", "p.jpg", img(n)} }

func TestHandlerMultipartCreate(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)

	rec := a.multipart(http.MethodPost, "/products", tok, baseFields(), []formFile{photo(1), photo(2)})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	got := decode[Product](t, rec)
	if got.Title != "Phone X" || got.Price != 1_000_000 || got.Stock != 10 || got.Status != StatusDraft || len(got.Photos) != 2 {
		t.Fatalf("created = %+v", got)
	}

	plain := a.multipart(http.MethodPost, "/products", tok, append(baseFields()[:4:4], [2]string{"sku", "PX-2"}), nil)
	if plain.Code != http.StatusBadRequest || errorCode(t, plain) != "validation_error" {
		t.Fatalf("no photos: %d %s", plain.Code, plain.Body)
	}
}

func TestHandlerMultipartErrors(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	with := func(extra ...[2]string) [][2]string { return append(baseFields(), extra...) }
	big := formFile{"photos", "big.jpg", bytes.Repeat([]byte{1}, maxPhotoBytes+1)}
	six := []formFile{photo(1), photo(2), photo(3), photo(4), photo(5), photo(6)}

	cases := []struct {
		name   string
		fields [][2]string
		files  []formFile
		status int
		code   string
	}{
		{"price is not a number", [][2]string{{"category_id", "1"}, {"title", "T"}, {"price", "abc"}, {"sku", "S"}}, nil, 400, "validation_error"},
		{"missing title", [][2]string{{"category_id", "1"}, {"price", "5"}, {"sku", "S"}}, nil, 400, "validation_error"},
		{"unknown field", with([2]string{"color", "red"}), nil, 400, "validation_error"},
		{"status on create", with([2]string{"status", "active"}), nil, 400, "validation_error"},
		{"keep_photos on create", with([2]string{"keep_photos", "x"}), nil, 400, "validation_error"},
		{"field sent twice", with([2]string{"title", "Again"}), nil, 400, "validation_error"},
		{"file in a wrong field", baseFields(), []formFile{{"image", "a.jpg", img(1)}}, 400, "validation_error"},
		{"empty file", baseFields(), []formFile{{"photos", "a.jpg", nil}}, 400, "validation_error"},
		{"six photos", baseFields(), six, 400, "validation_error"},
		{"photo over 5 MB", baseFields(), []formFile{big}, 413, "file_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := a.multipart(http.MethodPost, "/products", tok, tc.fields, tc.files)
			if rec.Code != tc.status || errorCode(t, rec) != tc.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
		})
	}
	if len(a.photos.live()) != 0 {
		t.Fatalf("refused requests left files: %v", a.photos.live())
	}

	t.Run("photo store errors map to statuses", func(t *testing.T) {
		for _, c := range []struct {
			err    error
			status int
			code   string
		}{
			{fmt.Errorf("%w: only jpeg", ErrUnsupportedMedia), 415, "unsupported_media_type"},
			{ErrFileTooLarge, 413, "file_too_large"},
			{ErrBusy, 503, "busy"},
			{fmt.Errorf("%w: not an image", ErrValidation), 400, "validation_error"},
			{errors.New("disk full"), 500, "internal"},
		} {
			a.photos.failAt, a.photos.failErr = a.photos.next+1, c.err
			rec := a.multipart(http.MethodPost, "/products", tok, baseFields(), []formFile{photo(1)})
			if rec.Code != c.status || errorCode(t, rec) != c.code {
				t.Errorf("%v: got %d %s, want %d %s", c.err, rec.Code, rec.Body, c.status, c.code)
			}
			if c.status == 503 && rec.Header().Get("Retry-After") == "" {
				t.Error("503 without Retry-After")
			}
			if c.status == 500 && strings.Contains(rec.Body.String(), "disk full") {
				t.Error("500 leaked the cause")
			}
		}
		a.photos.failAt = 0
	})

	t.Run("requires a token", func(t *testing.T) {
		rec := a.multipart(http.MethodPost, "/products", "", baseFields(), []formFile{photo(1)})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})
	t.Run("truncated body", func(t *testing.T) {
		body, ct := multipartBody(t, baseFields(), []formFile{photo(1)})
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body.Bytes()[:body.Len()/2]))
		req.Header.Set("Content-Type", ct)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		a.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body)
		}
	})
}

func TestHandlerJSONCreateIsRefused(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	if rec := a.do(http.MethodPost, "/products", tok, createBody); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "validation_error" {
		t.Fatalf("a product without photos: %d %s", rec.Code, rec.Body)
	}
	withPhotos := strings.Replace(createBody, `"stock":10`, `"stock":10,"photos":["/x.webp"]`, 1)
	if rec := a.do(http.MethodPost, "/products", tok, withPhotos); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_body" {
		t.Fatalf("a photos list in JSON: %d %s", rec.Code, rec.Body)
	}
}

func TestHandlerMultipartUpdate(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	created := decode[Product](t, a.multipart(http.MethodPost, "/products", tok, baseFields(), []formFile{photo(1), photo(2)}))
	path := "/products/" + strconv.FormatInt(created.ID, 10)
	fields := func(extra ...[2]string) [][2]string {
		return append([][2]string{{"category_id", "1"}, {"title", "Phone Z"}, {"price", "2000000"}, {"sku", "PX-1"}}, extra...)
	}

	t.Run("no keep_photos keeps all and adds", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields(), []formFile{photo(3)})
		got := decode[Product](t, rec)
		if rec.Code != 200 || len(got.Photos) != 3 || got.Photos[0] != created.Photos[0] || got.Title != "Phone Z" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		created = got
	})
	t.Run("keep_photos picks the survivors", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"keep_photos", created.Photos[2]}, [2]string{"keep_photos", created.Photos[0]}), nil)
		got := decode[Product](t, rec)
		if rec.Code != 200 || len(got.Photos) != 2 || got.Photos[0] != created.Photos[2] || got.Photos[1] != created.Photos[0] {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		created = got
	})
	t.Run("an empty keep_photos without a new file is refused", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"keep_photos", ""}), nil)
		if rec.Code != 400 || errorCode(t, rec) != "validation_error" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("keep_photos with a foreign url", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"keep_photos", "https://evil.example.com/a.webp"}), nil)
		if rec.Code != 400 || errorCode(t, rec) != "validation_error" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("status changes the status", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"status", "active"}), nil)
		if got := decode[Product](t, rec); rec.Code != 200 || got.Status != StatusActive {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("status cannot archive", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"status", "archived"}), nil)
		if rec.Code != 400 || errorCode(t, rec) != "validation_error" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("stock is not an update field", func(t *testing.T) {
		rec := a.multipart(http.MethodPut, path, tok, fields([2]string{"stock", "5"}), nil)
		if rec.Code != 400 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("another shop gets not found and stores nothing", func(t *testing.T) {
		before := len(a.photos.live())
		rec := a.multipart(http.MethodPut, path, a.token(2), fields(), []formFile{photo(7)})
		if rec.Code != 404 || errorCode(t, rec) != "not_found" || len(a.photos.live()) != before {
			t.Fatalf("%d %s, live files %d -> %d", rec.Code, rec.Body, before, len(a.photos.live()))
		}
	})
}
