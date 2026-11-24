package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

func newOTPHandler(t *testing.T, tg Telegram) (*Handler, *memRepository, *TokenIssuer) {
	t.Helper()
	repo := newMemRepository()
	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(repo, tokens, tg)
	if err != nil {
		t.Fatal(err)
	}
	guard := httpx.NewLimiterPer(100, time.Minute)
	return NewHandler(svc, guard, func(*http.Request) string { return "203.0.113.5" }), repo, tokens
}

func postJSON(t *testing.T, h http.HandlerFunc, body any, principal *Principal) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	if principal != nil {
		req = req.WithContext(context.WithValue(req.Context(), principalKey{}, *principal))
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestRequestCodeAnswersTheSameForAnyEmail(t *testing.T) {
	tg := &fakeTelegram{}
	h, repo, _ := newOTPHandler(t, tg)
	sellerID, _, err := repo.CreateSellerWithShop(context.Background(), "seller@example.com", "hash", "Shop")
	if err != nil {
		t.Fatal(err)
	}
	repo.chats[sellerID] = linkedChat

	for _, email := range []string{"seller@example.com", "nobody@example.com"} {
		rec := postJSON(t, h.RequestCode, RequestCodeRequest{Email: email}, nil)
		if rec.Code != http.StatusAccepted || rec.Body.String() != `{"status":"sent"}`+"\n" {
			t.Fatalf("%s: %d %s", email, rec.Code, rec.Body.String())
		}
	}
}

func TestRequestCodeUnavailableWithoutTelegram(t *testing.T) {
	h, _, _ := newOTPHandler(t, nil)
	rec := postJSON(t, h.RequestCode, RequestCodeRequest{Email: "a@example.com"}, nil)
	assertError(t, rec, http.StatusServiceUnavailable, "telegram_unavailable")
}

func TestRequestCodeRejectsBadBody(t *testing.T) {
	h, _, _ := newOTPHandler(t, &fakeTelegram{})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@example.com","extra":1}`))
	rec := httptest.NewRecorder()
	h.RequestCode(rec, req)
	assertError(t, rec, http.StatusBadRequest, "invalid_body")
}

func TestVerifyWrongCodeIsUnauthorized(t *testing.T) {
	tg := &fakeTelegram{}
	h, repo, _ := newOTPHandler(t, tg)
	sellerID, _, _ := repo.CreateSellerWithShop(context.Background(), "seller@example.com", "hash", "Shop")
	repo.chats[sellerID] = linkedChat
	postJSON(t, h.RequestCode, RequestCodeRequest{Email: "seller@example.com"}, nil)

	rec := postJSON(t, h.Verify, VerifyRequest{Email: "seller@example.com", Code: "000000"}, nil)
	if tg.lastCode(t) == "000000" {
		rec = postJSON(t, h.Verify, VerifyRequest{Email: "seller@example.com", Code: "111111"}, nil)
	}
	assertError(t, rec, http.StatusUnauthorized, "invalid_code")
}

func TestVerifyCorrectCodeReturnsToken(t *testing.T) {
	tg := &fakeTelegram{}
	h, repo, tokens := newOTPHandler(t, tg)
	sellerID, shopID, _ := repo.CreateSellerWithShop(context.Background(), "seller@example.com", "hash", "Shop")
	repo.chats[sellerID] = linkedChat
	postJSON(t, h.RequestCode, RequestCodeRequest{Email: "seller@example.com"}, nil)

	rec := postJSON(t, h.Verify, VerifyRequest{Email: "seller@example.com", Code: tg.lastCode(t)}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	var res TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	claims, err := tokens.Parse(res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SellerID != sellerID || claims.ShopID != shopID {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestLinkTelegramNeedsPrincipal(t *testing.T) {
	h, _, _ := newOTPHandler(t, &fakeTelegram{})
	rec := postJSON(t, h.LinkTelegram, struct{}{}, nil)
	assertError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestLinkTelegramReturnsLink(t *testing.T) {
	h, _, _ := newOTPHandler(t, &fakeTelegram{})
	rec := postJSON(t, h.LinkTelegram, struct{}{}, &Principal{SellerID: 1, ShopID: 1})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	var res TelegramLinkResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.URL == "" || res.ExpiresAt.IsZero() {
		t.Fatalf("response = %+v", res)
	}
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d, body %s", rec.Code, status, rec.Body.String())
	}
	var body httpx.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != code {
		t.Fatalf("body = %s, want code %q", rec.Body.String(), code)
	}
}
