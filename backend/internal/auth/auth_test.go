package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

func newService(t *testing.T) (*Service, *TokenIssuer) {
	t.Helper()
	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(newMemRepository(), tokens, nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, tokens
}

func register(t *testing.T, svc *Service, email string) TokenResponse {
	t.Helper()
	res, err := svc.Register(context.Background(), RegisterRequest{
		Email: email, Password: "correct-horse", ShopName: "Shop " + email,
	})
	if err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
	return res
}

func TestRegisterThenLogin(t *testing.T) {
	svc, tokens := newService(t)
	reg := register(t, svc, "seller@example.com")

	claims, err := tokens.Parse(reg.Token)
	if err != nil {
		t.Fatalf("register token does not parse: %v", err)
	}
	if claims.SellerID == 0 || claims.ShopID == 0 {
		t.Fatalf("claims missing ids: %+v", claims)
	}

	login, err := svc.Login(context.Background(), LoginRequest{Email: "seller@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	loginClaims, _ := tokens.Parse(login.Token)
	if loginClaims.SellerID != claims.SellerID || loginClaims.ShopID != claims.ShopID {
		t.Fatalf("login ids differ from register ids")
	}
}

func TestEmailIsNormalised(t *testing.T) {
	svc, _ := newService(t)
	register(t, svc, "  Seller@Example.COM ")
	if _, err := svc.Login(context.Background(), LoginRequest{Email: "seller@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("login with lower-case email failed: %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _ := newService(t)
	register(t, svc, "dup@example.com")
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "DUP@example.com", Password: "correct-horse", ShopName: "Other",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _ := newService(t)
	cases := map[string]RegisterRequest{
		"bad email":      {Email: "not-an-email", Password: "correct-horse", ShopName: "S"},
		"display name":   {Email: "Name <a@example.com>", Password: "correct-horse", ShopName: "S"},
		"short password": {Email: "a@example.com", Password: "short", ShopName: "S"},
		"long password":  {Email: "a@example.com", Password: strings.Repeat("x", 73), ShopName: "S"},
		"empty shop":     {Email: "a@example.com", Password: "correct-horse", ShopName: "   "},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), req)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestLoginWrongPasswordAndUnknownEmailLookTheSame(t *testing.T) {
	svc, _ := newService(t)
	register(t, svc, "a@example.com")
	ctx := context.Background()

	_, errWrong := svc.Login(ctx, LoginRequest{Email: "a@example.com", Password: "wrong-password"})
	_, errUnknown := svc.Login(ctx, LoginRequest{Email: "nobody@example.com", Password: "wrong-password"})
	if !errors.Is(errWrong, ErrInvalidCredentials) || !errors.Is(errUnknown, ErrInvalidCredentials) {
		t.Fatalf("errors = %v / %v, want ErrInvalidCredentials for both", errWrong, errUnknown)
	}
}

func TestPasswordIsNotStoredInPlainText(t *testing.T) {
	store := newMemRepository()
	tokens, _ := NewTokenIssuer("s", time.Hour)
	svc, _ := NewService(store, tokens, nil)
	register(t, svc, "a@example.com")
	if c := store.byMail["a@example.com"]; c.PasswordHash == "correct-horse" || c.PasswordHash == "" {
		t.Fatalf("stored password hash looks wrong: %q", c.PasswordHash)
	}
}

func TestTokenRejectsWrongSecretAndExpiry(t *testing.T) {
	issuer, _ := NewTokenIssuer("secret-a", time.Hour)
	token, _, err := issuer.Issue(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	other, _ := NewTokenIssuer("secret-b", time.Hour)
	if _, err := other.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong secret: err = %v, want ErrInvalidToken", err)
	}

	expired, _ := NewTokenIssuer("secret-a", time.Hour)
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	past, _, _ := expired.Issue(1, 1)
	if _, err := issuer.Parse(past); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired: err = %v, want ErrInvalidToken", err)
	}
}

func TestTokenRejectsOtherAlgorithms(t *testing.T) {
	// "none" algorithm token with no signature must be refused.
	issuer, _ := NewTokenIssuer("s", time.Hour)
	noneToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzZWxsZXJfaWQiOjEsInNob3BfaWQiOjF9."
	if _, err := issuer.Parse(noneToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("alg none: err = %v, want ErrInvalidToken", err)
	}
}

func TestNewTokenIssuerRequiresSecret(t *testing.T) {
	if _, err := NewTokenIssuer("", time.Hour); err == nil {
		t.Fatal("empty secret accepted")
	}
}

func newTestHandler(svc *Service, guard LoginGuard) *Handler {
	return NewHandler(svc, guard, func(r *http.Request) string { return r.RemoteAddr })
}

func TestHandlerRegisterAndLoginStatusCodes(t *testing.T) {
	svc, _ := newService(t)
	h := newTestHandler(svc, httpx.NewLimiterPer(5, time.Hour))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)

	do := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		return rec
	}

	reg := do("/auth/register", `{"email":"h@example.com","password":"correct-horse","shop_name":"Shop"}`)
	if reg.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", reg.Code)
	}
	var tok TokenResponse
	if err := json.Unmarshal(reg.Body.Bytes(), &tok); err != nil || tok.Token == "" {
		t.Fatalf("register body has no token: %v", err)
	}

	if rec := do("/auth/register", `{"email":"h@example.com","password":"correct-horse","shop_name":"Shop"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", rec.Code)
	}
	if rec := do("/auth/register", `{"email":"x@example.com","password":"correct-horse","shop_name":"S","extra":1}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", rec.Code)
	}
	if rec := do("/auth/login", `{"email":"h@example.com","password":"wrong-password"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", rec.Code)
	}
	if rec := do("/auth/login", `{"email":"h@example.com","password":"correct-horse"}`); rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", rec.Code)
	}
}

func TestRequireAuthScopesEachRequestToItsSeller(t *testing.T) {
	svc, tokens := newService(t)
	a := register(t, svc, "a@example.com")
	b := register(t, svc, "b@example.com")

	var got []Principal
	protected := RequireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			t.Error("principal missing in context")
		}
		got = append(got, p)
		w.WriteHeader(http.StatusOK)
	}))

	for _, tok := range []string{a.Token, b.Token} {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		protected.ServeHTTP(httptest.NewRecorder(), req)
	}
	if len(got) != 2 || got[0].SellerID == got[1].SellerID || got[0].ShopID == got[1].ShopID {
		t.Fatalf("principals not separated per seller: %+v", got)
	}
}

func TestRequireAuthRejectsMissingOrBadToken(t *testing.T) {
	_, tokens := newService(t)
	protected := RequireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cases := map[string]string{
		"no header":    "",
		"no bearer":    "Basic abc",
		"garbage":      "Bearer not-a-token",
		"empty bearer": "Bearer ",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestLoginIsBlockedAfterFiveFailuresPerIPAndEmail(t *testing.T) {
	svc, _ := newService(t)
	register(t, svc, "guard@example.com")
	register(t, svc, "other@example.com")
	h := newTestHandler(svc, httpx.NewLimiterPer(5, 15*time.Minute))

	login := func(addr, email, password string) *httptest.ResponseRecorder {
		body := `{"email":"` + email + `","password":"` + password + `"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec
	}

	for i := 1; i <= 5; i++ {
		if rec := login("1.1.1.1:1000", "guard@example.com", "wrong-password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d, want 401", i, rec.Code)
		}
	}
	rec := login("1.1.1.1:1000", "guard@example.com", "correct-horse")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after 5 failures even the right password gets %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing")
	}
	var e struct{ Code string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != "rate_limited" {
		t.Fatalf("body = %q, want code rate_limited", rec.Body.String())
	}

	if rec := login("1.1.1.1:1000", "GUARD@example.com ", "correct-horse"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("email case and spaces must not dodge the limit, got %d", rec.Code)
	}
	if rec := login("1.1.1.1:1000", "other@example.com", "correct-horse"); rec.Code != http.StatusOK {
		t.Fatalf("another email from the same IP got %d, want 200", rec.Code)
	}
	if rec := login("2.2.2.2:2000", "guard@example.com", "correct-horse"); rec.Code != http.StatusOK {
		t.Fatalf("same email from another IP got %d, want 200", rec.Code)
	}
}

func TestSuccessfulLoginResetsFailures(t *testing.T) {
	svc, _ := newService(t)
	register(t, svc, "reset@example.com")
	h := newTestHandler(svc, httpx.NewLimiterPer(3, time.Hour))

	login := func(password string) int {
		body := `{"email":"reset@example.com","password":"` + password + `"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.RemoteAddr = "3.3.3.3:3000"
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		return rec.Code
	}
	for round := 0; round < 3; round++ {
		for i := 0; i < 2; i++ {
			if code := login("wrong-password"); code != http.StatusUnauthorized {
				t.Fatalf("round %d: wrong password got %d", round, code)
			}
		}
		if code := login("correct-horse"); code != http.StatusOK {
			t.Fatalf("round %d: right password got %d, a success must reset the counter", round, code)
		}
	}
}
