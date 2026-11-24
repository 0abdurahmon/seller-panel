package shop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

type multipartAPI struct {
	t     *testing.T
	mux   *http.ServeMux
	store *fakeLogoStore
	repo  *memRepository
	token string
}

func newMultipartAPI(t *testing.T) *multipartAPI {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	repo := newMemRepository()
	repo.seed(1, "Shop A")
	repo.seed(2, "Shop B")
	store := &fakeLogoStore{}
	h := NewHandler(NewService(repo, store))
	mux := http.NewServeMux()
	mux.Handle("GET /shop", auth.RequireAuth(tokens)(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /shop", auth.RequireAuth(tokens)(http.HandlerFunc(h.Update)))
	token, _, _ := tokens.Issue(10, 1)
	return &multipartAPI{t: t, mux: mux, store: store, repo: repo, token: token}
}

type part struct {
	field string
	value string
	file  []byte
}

func (a *multipartAPI) put(token string, parts ...part) *httptest.ResponseRecorder {
	a.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range parts {
		if p.file != nil {
			fw, err := mw.CreateFormFile(p.field, "logo.png")
			if err != nil {
				a.t.Fatal(err)
			}
			fw.Write(p.file)
			continue
		}
		mw.WriteField(p.field, p.value)
	}
	mw.Close()
	return a.send(token, mw.FormDataContentType(), &body)
}

func (a *multipartAPI) send(token, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/shop", body)
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec
}

func fields(extra ...part) []part {
	return append([]part{{field: "name", value: "New name"}, {field: "description", value: "About"}, {field: "contact_phone", value: "+998 90 123 45 67"}}, extra...)
}

func shopOf(t *testing.T, rec *httptest.ResponseRecorder) Shop {
	t.Helper()
	var s Shop
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("%v: %s", err, rec.Body)
	}
	return s
}

func code(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code == "" || e.Message == "" {
		t.Fatalf("not an error body: %s", rec.Body)
	}
	return e.Code
}

func TestMultipartUpdateWithLogo(t *testing.T) {
	a := newMultipartAPI(t)
	rec := a.put(a.token, fields(part{field: "logo", file: []byte("image"), value: ""})...)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	got := shopOf(t, rec)
	if got.Name != "New name" || got.ID != 1 || got.ContactPhone == nil || *got.ContactPhone != "+998901234567" || got.LogoURL == nil {
		t.Fatalf("shop = %+v", got)
	}

	first := *got.LogoURL
	rec = a.put(a.token, fields(part{field: "logo", file: []byte("another")})...)
	if rec.Code != 200 || len(a.store.removed) != 1 || a.store.removed[0] != first {
		t.Fatalf("replace: %d, removed %v, want the old logo %q", rec.Code, a.store.removed, first)
	}

	rec = a.put(a.token, fields(part{field: "remove_logo", value: "true"})...)
	if rec.Code != 200 || shopOf(t, rec).LogoURL != nil || !strings.Contains(rec.Body.String(), `"logo_url":null`) {
		t.Fatalf("remove: %d %s", rec.Code, rec.Body)
	}

	rec = a.put(a.token, fields(part{field: "remove_logo", value: "false"})...)
	if rec.Code != 200 {
		t.Fatalf("remove_logo=false: %d %s", rec.Code, rec.Body)
	}
}

func TestMultipartUpdateErrors(t *testing.T) {
	a := newMultipartAPI(t)
	big := part{field: "logo", file: bytes.Repeat([]byte{1}, maxLogoBytes+1)}
	cases := []struct {
		name   string
		parts  []part
		status int
		code   string
	}{
		{"missing name", []part{{field: "description", value: "x"}}, 400, "validation_error"},
		{"unknown field", fields(part{field: "logo_url", value: "https://example.com/a.png"}), 400, "validation_error"},
		{"field sent twice", fields(part{field: "name", value: "Again"}), 400, "validation_error"},
		{"file in a wrong field", fields(part{field: "avatar", file: []byte("x")}), 400, "validation_error"},
		{"two logos", fields(part{field: "logo", file: []byte("a")}, part{field: "logo", file: []byte("b")}), 400, "validation_error"},
		{"empty logo", fields(part{field: "logo", file: []byte{}, value: ""}), 400, "validation_error"},
		{"remove_logo is not a boolean", fields(part{field: "remove_logo", value: "maybe"}), 400, "validation_error"},
		{"a logo and remove_logo", fields(part{field: "logo", file: []byte("x")}, part{field: "remove_logo", value: "true"}), 400, "validation_error"},
		{"logo over 5 MB", fields(big), 413, "file_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := a.put(a.token, tc.parts...)
			if rec.Code != tc.status || code(t, rec) != tc.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
		})
	}
	if len(a.store.saved) != 0 {
		t.Fatalf("refused requests stored %v", a.store.saved)
	}
	if got, _ := NewService(a.repo, a.store).Get(t.Context(), 1); got.Name != "Shop A" {
		t.Fatalf("a refused request changed the shop: %+v", got)
	}

	t.Run("store errors map to statuses", func(t *testing.T) {
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
			a.store.failErr = c.err
			rec := a.put(a.token, fields(part{field: "logo", file: []byte("x")})...)
			if rec.Code != c.status || code(t, rec) != c.code {
				t.Errorf("%v: got %d %s, want %d %s", c.err, rec.Code, rec.Body, c.status, c.code)
			}
			if c.status == 503 && rec.Header().Get("Retry-After") == "" {
				t.Error("503 without Retry-After")
			}
			if c.status == 500 && strings.Contains(rec.Body.String(), "disk full") {
				t.Error("500 leaked the cause")
			}
		}
		a.store.failErr = nil
	})

	t.Run("requires a token", func(t *testing.T) {
		if rec := a.put("", fields()...); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})
	t.Run("truncated body", func(t *testing.T) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		mw.WriteField("name", "X")
		fw, _ := mw.CreateFormFile("logo", "l.png")
		fw.Write([]byte("image"))
		mw.Close()
		cut := bytes.NewBuffer(body.Bytes()[:body.Len()/2])
		if rec := a.send(a.token, mw.FormDataContentType(), cut); rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body)
		}
	})
}

func TestMultipartLogoIsScopedToTheTokensShop(t *testing.T) {
	a := newMultipartAPI(t)
	rec := a.put(a.token, fields(part{field: "logo", file: []byte("x")})...)
	got := shopOf(t, rec)
	if got.ID != 1 || !strings.HasPrefix(*got.LogoURL, "/api/uploads/1/") {
		t.Fatalf("shop = %+v, the logo must be stored for shop 1", got)
	}
	if other, _ := NewService(a.repo, a.store).Get(t.Context(), 2); other.Name != "Shop B" || other.LogoURL != nil {
		t.Fatalf("shop 2 changed: %+v", other)
	}
}
