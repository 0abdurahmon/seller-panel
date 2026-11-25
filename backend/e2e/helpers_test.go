//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/app"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

const jwtSecret = "e2e-secret"

var counter atomic.Int64

type server struct {
	t    *testing.T
	url  string
	pool *pgxpool.Pool
	app  *app.App
	dir  string
}

func newServer(t *testing.T, mutate ...func(*config.Config)) *server {
	t.Helper()
	pool := dbtest.NewPool(t, "e2e")
	zone, err := time.LoadLocation(config.DefaultBusinessTimeZone)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := config.Config{
		Env:               config.EnvDevelopment,
		OpenAPIFile:       "../../api/openapi.yaml",
		Port:              "0",
		JWTSecret:         jwtSecret,
		JWTTTL:            time.Hour,
		CommissionPercent: 10,
		ShopRPS:           1000,
		ShopBurst:         1000,
		LoginAttempts:     5,
		LoginWindow:       15 * time.Minute,
		TrustedProxies:    []string{"127.0.0.1/32", "::1/128"},
		BusinessLocation:  zone,
		UploadDir:         dir,
		UploadPublicURL:   "/api/uploads",
		ImagesPerMinute:   1000,
	}
	for _, m := range mutate {
		m(&cfg)
	}
	application, err := app.NewApp(cfg, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	// Cleanups run last in, first out: the server stops, then the upload directory is released, then the temp directory goes.
	t.Cleanup(func() { _ = application.Close() })
	srv := httptest.NewServer(application.Handler)
	t.Cleanup(srv.Close)
	return &server{t: t, url: srv.URL, pool: pool, app: application, dir: dir}
}

type response struct {
	t      *testing.T
	status int
	header http.Header
	body   []byte
}

func (r response) decode(v any) {
	r.t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		r.t.Fatalf("body is not the expected JSON: %v: %s", err, r.body)
	}
}

func (r response) errorCode() string {
	r.t.Helper()
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	r.decode(&e)
	if e.Code == "" || e.Message == "" {
		r.t.Fatalf("error body must carry code and message, got %s", r.body)
	}
	return e.Code
}

func (s *server) do(method, path, token string, body any, headers ...string) response {
	s.t.Helper()
	res, err := s.tryDo(method, path, token, body, headers...)
	if err != nil {
		s.t.Fatal(err)
	}
	return res
}

func (s *server) tryDo(method, path, token string, body any, headers ...string) (response, error) {
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return response{}, err
		}
		reader = bytes.NewBuffer(raw)
	}
	req, err := http.NewRequest(method, s.url+path, reader)
	if err != nil {
		return response{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return response{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return response{}, err
	}
	return response{t: s.t, status: res.StatusCode, header: res.Header, body: raw}, nil
}

type seller struct {
	email    string
	password string
	shopName string
	token    string
}

func uniqueEmail() string {
	return fmt.Sprintf("e2e-%d-%d@example.com", time.Now().UnixNano(), counter.Add(1))
}

func (s *server) register(headers ...string) seller {
	s.t.Helper()
	sel := seller{email: uniqueEmail(), password: "correct-horse", shopName: fmt.Sprintf("Shop %d", counter.Add(1))}
	res := s.do(http.MethodPost, "/auth/register", "", map[string]string{
		"email": sel.email, "password": sel.password, "shop_name": sel.shopName,
	}, headers...)
	if res.status != http.StatusCreated {
		s.t.Fatalf("register status = %d: %s", res.status, res.body)
	}
	var tok struct {
		Token string `json:"token"`
	}
	res.decode(&tok)
	sel.token = tok.Token
	return sel
}

func (s *server) login(email, password string, headers ...string) response {
	s.t.Helper()
	return s.do(http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": password}, headers...)
}

type product struct {
	ID          int64    `json:"id"`
	ShopID      int64    `json:"shop_id"`
	CategoryID  int64    `json:"category_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	SKU         string   `json:"sku"`
	Stock       int      `json:"stock"`
	Reserved    int      `json:"reserved"`
	Status      string   `json:"status"`
	Photos      []string `json:"photos"`
}

type productPage struct {
	Items    []product `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

func (s *server) categoryID(token string) int64 {
	s.t.Helper()
	res := s.do(http.MethodGet, "/categories", token, nil)
	var list struct {
		Items []struct {
			ID       int64  `json:"id"`
			ParentID *int64 `json:"parent_id"`
		} `json:"items"`
	}
	res.decode(&list)
	for _, c := range list.Items {
		if c.ParentID != nil {
			return c.ID
		}
	}
	s.t.Fatal("no child category in the seed")
	return 0
}

func productBody(category int64, sku string, stock int) map[string]any {
	return map[string]any{
		"category_id": category, "title": "Phone " + sku, "description": "Good phone",
		"price": 1_500_000, "sku": sku, "stock": stock,
	}
}

// postProduct creates a product from the fields, with one small photo.
func (s *server) postProduct(token string, fields map[string]any) response {
	s.t.Helper()
	parts := []formPart{photoPart(pngFixture(s.t, 10, 10))}
	for name, v := range fields {
		parts = append(parts, text(name, fmt.Sprint(v)))
	}
	return s.productForm(http.MethodPost, "/products", token, parts...)
}

func uniqueSKU() string {
	return fmt.Sprintf("SKU-%d-%d", time.Now().UnixNano(), counter.Add(1))
}

func (s *server) createProduct(token string, category int64, stock int) product {
	s.t.Helper()
	res := s.postProduct(token, productBody(category, uniqueSKU(), stock))
	if res.status != http.StatusCreated {
		s.t.Fatalf("create product status = %d: %s", res.status, res.body)
	}
	var p product
	res.decode(&p)
	return s.activate(token, p)
}

// activate moves a draft product to active: a product is created as a draft, so tests that need one
// to be sold (simulate-order only picks active products) publish it with an edit.
func (s *server) activate(token string, p product) product {
	s.t.Helper()
	body := productBody(p.CategoryID, p.SKU, 0)
	delete(body, "stock")
	body["status"] = "active"
	res := s.do(http.MethodPut, productPath(p), token, body)
	if res.status != http.StatusOK {
		s.t.Fatalf("activate product status = %d: %s", res.status, res.body)
	}
	res.decode(&p)
	return p
}

func productPath(p product) string {
	return fmt.Sprintf("/products/%d", p.ID)
}

func (s *server) expect(res response, status int, code string) {
	s.t.Helper()
	if res.status != status {
		s.t.Fatalf("status = %d, want %d: %s", res.status, status, res.body)
	}
	if code != "" {
		if got := res.errorCode(); got != code {
			s.t.Fatalf("error code = %q, want %q: %s", got, code, res.body)
		}
	}
}
