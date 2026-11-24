package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func liveShopHandler(check ShopChecker, ttl time.Duration, c *clock) http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return requireLiveShop(check, ttl, c.now)(next)
}

func callWith(h http.Handler, p Principal) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), principalKey{}, p))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRequireLiveShopLetsAnExistingShopIn(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	calls := 0
	h := liveShopHandler(func(_ context.Context, seller, shop int64) (bool, error) {
		calls++
		return seller == 1 && shop == 2, nil
	}, time.Minute, c)

	for i := 0; i < 3; i++ {
		if rec := callWith(h, Principal{SellerID: 1, ShopID: 2}); rec.Code != http.StatusNoContent {
			t.Fatalf("call %d: status %d", i, rec.Code)
		}
	}
	if calls != 1 {
		t.Fatalf("the check ran %d times, want once while the answer is remembered", calls)
	}
	c.t = c.t.Add(2 * time.Minute)
	callWith(h, Principal{SellerID: 1, ShopID: 2})
	if calls != 2 {
		t.Fatalf("the check ran %d times, want it again after the memory expired", calls)
	}
}

func TestRequireLiveShopRefusesAShopThatIsGone(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	exists := true
	h := liveShopHandler(func(context.Context, int64, int64) (bool, error) { return exists, nil }, time.Hour, c)
	p := Principal{SellerID: 1, ShopID: 1}

	if rec := callWith(h, p); rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	exists = false
	if rec := callWith(h, p); rec.Code != http.StatusNoContent {
		t.Fatalf("a remembered shop is trusted until the memory expires, status %d", rec.Code)
	}
	c.t = c.t.Add(2 * time.Hour)
	rec := callWith(h, p)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("a gone shop: %d %s", rec.Code, rec.Body)
	}
	exists = true
	if rec := callWith(h, p); rec.Code != http.StatusNoContent {
		t.Fatalf("a refusal must not be remembered, status %d", rec.Code)
	}
}

func TestRequireLiveShopRefusesAShopOfAnotherSeller(t *testing.T) {
	h := liveShopHandler(func(_ context.Context, seller, shop int64) (bool, error) { return seller == 5 && shop == 7, nil }, time.Minute, &clock{t: time.Unix(1, 0)})
	if rec := callWith(h, Principal{SellerID: 6, ShopID: 7}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("a token of seller 6 for the shop of seller 5: status %d", rec.Code)
	}
}

func TestRequireLiveShopAnswers500WhenTheCheckFails(t *testing.T) {
	h := liveShopHandler(func(context.Context, int64, int64) (bool, error) { return false, errors.New("database is down") }, time.Minute, &clock{t: time.Unix(1, 0)})
	rec := callWith(h, Principal{SellerID: 1, ShopID: 1})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500: a failing check must not let the request in", rec.Code)
	}
	if body := rec.Body.String(); body == "" || strings.Contains(body, "database is down") {
		t.Fatalf("body = %q, the cause must not leak", body)
	}
}

func TestRequireLiveShopNeedsAPrincipal(t *testing.T) {
	h := liveShopHandler(func(context.Context, int64, int64) (bool, error) { return true, nil }, time.Minute, &clock{t: time.Unix(1, 0)})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
