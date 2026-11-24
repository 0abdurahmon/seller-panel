package upload

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

type fixture struct {
	mux  *http.ServeMux
	repo *memRepository
	tokA string
	tokB string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	repo := newMemRepository()
	h := NewHandler(NewService(repo, "/api/uploads"))
	mux := http.NewServeMux()
	mux.Handle("POST /uploads", auth.RequireAuth(tokens)(http.HandlerFunc(h.Create)))
	a, _, _ := tokens.Issue(10, 1)
	b, _, _ := tokens.Issue(20, 2)
	return fixture{mux: mux, repo: repo, tokA: a, tokB: b}
}

func form(t *testing.T, field, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if field != "" {
		fw, err := mw.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(data)
	}
	mw.Close()
	return &body, mw.FormDataContentType()
}

func (f fixture) post(token, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/uploads", body)
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	f.mux.ServeHTTP(rec, req)
	return rec
}

func code(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code == "" || e.Message == "" {
		t.Fatalf("not an error body: %s", rec.Body)
	}
	return e.Code
}

func TestCreateSuccessShape(t *testing.T) {
	f := newFixture(t)
	body, ct := form(t, "file", "photo.jpg", jpegBytes(t, 100, 80))
	rec := f.post(f.tokA, ct, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var img Image
	if err := json.Unmarshal(rec.Body.Bytes(), &img); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(img.URL, "/api/uploads/1/") || img.ContentType != "image/webp" || img.Width != 100 || img.Height != 80 || img.Size <= 0 {
		t.Fatalf("unexpected body %s", rec.Body)
	}
}

func TestCreateErrors(t *testing.T) {
	f := newFixture(t)
	big := bytes.Repeat([]byte{1}, MaxFileBytes+1)
	tests := []struct {
		name   string
		token  string
		field  string
		data   []byte
		ctype  string
		status int
		code   string
	}{
		{"no token", "", "file", jpegBytes(t, 10, 10), "", http.StatusUnauthorized, "unauthorized"},
		{"bad token", "x.y.z", "file", jpegBytes(t, 10, 10), "", http.StatusUnauthorized, "unauthorized"},
		{"no file field", "A", "other", jpegBytes(t, 10, 10), "", http.StatusBadRequest, "validation_error"},
		{"empty form", "A", "", nil, "", http.StatusBadRequest, "validation_error"},
		{"empty file", "A", "file", []byte{}, "", http.StatusBadRequest, "validation_error"},
		{"not an image", "A", "file", []byte("hello"), "", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"gif", "A", "file", gifBytes(t), "", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"corrupt jpeg", "A", "file", jpegBytes(t, 100, 100)[:150], "", http.StatusBadRequest, "validation_error"},
		{"over 5 MB", "A", "file", big, "", http.StatusRequestEntityTooLarge, "file_too_large"},
		{"json body", "A", "", nil, "application/json", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"no content type", "A", "", nil, "none", http.StatusUnsupportedMediaType, "unsupported_media_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, ct := form(t, tc.field, "f.bin", tc.data)
			switch tc.ctype {
			case "":
			case "none":
				ct = ""
			default:
				ct = tc.ctype
			}
			token := tc.token
			if token == "A" {
				token = f.tokA
			}
			rec := f.post(token, ct, body)
			if rec.Code != tc.status || code(t, rec) != tc.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
		})
	}
	if f.repo.count() != 0 {
		t.Fatalf("failed uploads stored %d files", f.repo.count())
	}
}

func TestCreateTruncatedMultipart(t *testing.T) {
	f := newFixture(t)
	body, ct := form(t, "file", "a.png", pngBytes(t, 50, 50))
	cut := bytes.NewBuffer(body.Bytes()[:body.Len()/2])
	rec := f.post(f.tokA, ct, cut)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	code(t, rec)
}

func TestCreateIgnoresClientFilenameAndType(t *testing.T) {
	f := newFixture(t)
	body, ct := form(t, "file", "../../etc/passwd.php", pngBytes(t, 20, 20))
	rec := f.post(f.tokA, ct, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var img Image
	json.Unmarshal(rec.Body.Bytes(), &img)
	if strings.Contains(img.URL, "passwd") || strings.Contains(img.URL, "..") {
		t.Fatalf("client name leaked into %q", img.URL)
	}
}

func TestFilesAreNotServedByTheAPI(t *testing.T) {
	f := newFixture(t)
	body, ct := form(t, "file", "a.png", pngBytes(t, 20, 20))
	var img Image
	json.Unmarshal(f.post(f.tokA, ct, body).Body.Bytes(), &img)

	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, strings.TrimPrefix(img.URL, "/api"), nil))
	if rec.Code == http.StatusOK {
		t.Fatal("the handler must not serve files, nginx does")
	}
}
