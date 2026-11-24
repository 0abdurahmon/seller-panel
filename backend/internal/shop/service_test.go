package shop

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func validReq() UpdateRequest {
	return UpdateRequest{
		Name:          "Akbar Shop",
		LogoURL:       "https://cdn.example.com/logo.png",
		Description:   "Phones and cases",
		ContactPhone:  "+998 (90) 123-45-67",
		PickupAddress: "Chilonzor 7, Tashkent",
	}
}

func TestUpdateNormalisesAndStoresProfile(t *testing.T) {
	repo := NewMemRepository()
	repo.Seed(1, "Old")
	svc := NewService(repo)

	got, err := svc.Update(context.Background(), 1, validReq())
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "Akbar Shop" || got.Description != "Phones and cases" {
		t.Fatalf("name/description = %q / %q", got.Name, got.Description)
	}
	if got.ContactPhone == nil || *got.ContactPhone != "+998901234567" {
		t.Fatalf("contact_phone = %v, want +998901234567", got.ContactPhone)
	}
	if got.LogoURL == nil || got.PickupAddress == nil {
		t.Fatal("logo_url or pickup_address missing")
	}
}

func TestUpdateEmptyOptionalFieldsClearThem(t *testing.T) {
	repo := NewMemRepository()
	repo.Seed(1, "Old")
	svc := NewService(repo)
	if _, err := svc.Update(context.Background(), 1, validReq()); err != nil {
		t.Fatal(err)
	}

	req := validReq()
	req.LogoURL, req.ContactPhone, req.PickupAddress, req.Description = "", "", "", ""
	got, err := svc.Update(context.Background(), 1, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.LogoURL != nil || got.ContactPhone != nil || got.PickupAddress != nil {
		t.Fatalf("optional fields not cleared: %+v", got)
	}
	if got.Description != "" {
		t.Fatalf("description = %q, want empty string", got.Description)
	}
}

func TestUpdateValidation(t *testing.T) {
	long := strings.Repeat("a", 121)
	cases := map[string]func(*UpdateRequest){
		"empty name":        func(r *UpdateRequest) { r.Name = "   " },
		"long name":         func(r *UpdateRequest) { r.Name = long },
		"long description":  func(r *UpdateRequest) { r.Description = strings.Repeat("a", 2001) },
		"long address":      func(r *UpdateRequest) { r.PickupAddress = strings.Repeat("a", 301) },
		"letters in phone":  func(r *UpdateRequest) { r.ContactPhone = "+99890abc4567" },
		"phone too short":   func(r *UpdateRequest) { r.ContactPhone = "12345" },
		"phone too long":    func(r *UpdateRequest) { r.ContactPhone = "+1234567890123456" },
		"logo not a URL":    func(r *UpdateRequest) { r.LogoURL = "logo.png" },
		"logo wrong scheme": func(r *UpdateRequest) { r.LogoURL = "ftp://example.com/a.png" },
		"logo no host":      func(r *UpdateRequest) { r.LogoURL = "https:///a.png" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			repo := NewMemRepository()
			repo.Seed(1, "Old")
			req := validReq()
			mutate(&req)
			_, err := NewService(repo).Update(context.Background(), 1, req)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestUpdateUnknownShopIsNotFound(t *testing.T) {
	_, err := NewService(NewMemRepository()).Update(context.Background(), 42, validReq())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
