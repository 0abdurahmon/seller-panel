package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := auth.NewService(&fakeAuthRepository{byMail: map[string]auth.Credentials{}}, tokens)
	if err != nil {
		t.Fatal(err)
	}
	shopRepo := &fakeShopRepository{}
	return NewRouter(Deps{
		Auth:   auth.NewHandler(svc),
		Shop:   shop.NewHandler(shop.NewService(shopRepo)),
		Tokens: tokens,
	})
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

type fakeAuthRepository struct {
	byMail map[string]auth.Credentials
}

func (r *fakeAuthRepository) CreateSellerWithShop(_ context.Context, email, passwordHash, _ string) (int64, int64, error) {
	if _, taken := r.byMail[email]; taken {
		return 0, 0, auth.ErrEmailTaken
	}
	id := int64(len(r.byMail) + 1)
	r.byMail[email] = auth.Credentials{SellerID: id, ShopID: id, PasswordHash: passwordHash}
	return id, id, nil
}

func (r *fakeAuthRepository) FindCredentials(_ context.Context, email string) (auth.Credentials, error) {
	c, ok := r.byMail[email]
	if !ok {
		return auth.Credentials{}, auth.ErrNotFound
	}
	return c, nil
}

type fakeShopRepository struct{}

func (fakeShopRepository) Get(_ context.Context, shopID int64) (shop.Shop, error) {
	return shop.Shop{ID: shopID, Name: "Test shop"}, nil
}

func (fakeShopRepository) Update(_ context.Context, shopID int64, p shop.Profile) (shop.Shop, error) {
	return shop.Shop{ID: shopID, Name: p.Name}, nil
}
