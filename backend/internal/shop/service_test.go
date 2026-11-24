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
		Description:   "Phones and cases",
		ContactPhone:  "+998 (90) 123-45-67",
		PickupAddress: "Chilonzor 7, Tashkent",
	}
}

func TestUpdateNormalisesAndStoresProfile(t *testing.T) {
	repo := newMemRepository()
	repo.seed(1, "Old")
	svc := NewService(repo, &fakeLogoStore{})

	got, err := svc.Update(context.Background(), 1, validReq(), LogoChange{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "Akbar Shop" || got.Description != "Phones and cases" {
		t.Fatalf("name/description = %q / %q", got.Name, got.Description)
	}
	if got.ContactPhone == nil || *got.ContactPhone != "+998901234567" {
		t.Fatalf("contact_phone = %v, want +998901234567", got.ContactPhone)
	}
	if got.PickupAddress == nil {
		t.Fatal("pickup_address missing")
	}
}

func TestUpdateEmptyOptionalFieldsClearThem(t *testing.T) {
	repo := newMemRepository()
	repo.seed(1, "Old")
	svc := NewService(repo, &fakeLogoStore{})
	if _, err := svc.Update(context.Background(), 1, validReq(), LogoChange{}); err != nil {
		t.Fatal(err)
	}

	req := validReq()
	req.ContactPhone, req.PickupAddress, req.Description = "", "", ""
	got, err := svc.Update(context.Background(), 1, req, LogoChange{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContactPhone != nil || got.PickupAddress != nil {
		t.Fatalf("optional fields not cleared: %+v", got)
	}
	if got.Description != "" {
		t.Fatalf("description = %q, want empty string", got.Description)
	}
}

func TestUpdateValidation(t *testing.T) {
	long := strings.Repeat("a", 121)
	cases := map[string]func(*UpdateRequest){
		"empty name":       func(r *UpdateRequest) { r.Name = "   " },
		"long name":        func(r *UpdateRequest) { r.Name = long },
		"long description": func(r *UpdateRequest) { r.Description = strings.Repeat("a", 2001) },
		"long address":     func(r *UpdateRequest) { r.PickupAddress = strings.Repeat("a", 301) },
		"letters in phone": func(r *UpdateRequest) { r.ContactPhone = "+99890abc4567" },
		"phone too short":  func(r *UpdateRequest) { r.ContactPhone = "12345" },
		"phone too long":   func(r *UpdateRequest) { r.ContactPhone = "+1234567890123456" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			repo := newMemRepository()
			repo.seed(1, "Old")
			req := validReq()
			mutate(&req)
			_, err := NewService(repo, &fakeLogoStore{}).Update(context.Background(), 1, req, LogoChange{})
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestUpdateUnknownShopIsNotFound(t *testing.T) {
	_, err := NewService(newMemRepository(), &fakeLogoStore{}).Update(context.Background(), 42, validReq(), LogoChange{Add: []byte("x")})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func logoOf(s Shop) string {
	if s.LogoURL == nil {
		return ""
	}
	return *s.LogoURL
}

func TestUpdateLogo(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.seed(1, "Old")
	store := &fakeLogoStore{}
	svc := NewService(repo, store)

	got, err := svc.Update(ctx, 1, validReq(), LogoChange{Add: []byte("first")})
	if err != nil {
		t.Fatal(err)
	}
	first := logoOf(got)
	if first == "" || len(store.removed) != 0 {
		t.Fatalf("logo = %q, removed = %v", first, store.removed)
	}

	got, err = svc.Update(ctx, 1, validReq(), LogoChange{})
	if err != nil || logoOf(got) != first || len(store.removed) != 0 {
		t.Fatalf("an update without a logo change: logo %q (want %q), removed %v, err %v", logoOf(got), first, store.removed, err)
	}

	got, err = svc.Update(ctx, 1, validReq(), LogoChange{Add: []byte("second")})
	second := logoOf(got)
	if err != nil || second == "" || second == first {
		t.Fatalf("replace: logo %q, err %v", second, err)
	}
	if len(store.removed) != 1 || store.removed[0] != first {
		t.Fatalf("removed = %v, want the old logo %q", store.removed, first)
	}

	store.removed = nil
	got, err = svc.Update(ctx, 1, validReq(), LogoChange{Remove: true})
	if err != nil || got.LogoURL != nil {
		t.Fatalf("remove: logo %v, err %v", got.LogoURL, err)
	}
	if len(store.removed) != 1 || store.removed[0] != second {
		t.Fatalf("removed = %v, want %q", store.removed, second)
	}

	store.removed = nil
	if _, err := svc.Update(ctx, 1, validReq(), LogoChange{Remove: true}); err != nil || len(store.removed) != 0 {
		t.Fatalf("removing a missing logo: err %v, removed %v", err, store.removed)
	}
}

func TestUpdateLogoFailuresChangeNothing(t *testing.T) {
	ctx := context.Background()
	setup := func() (*memRepository, *fakeLogoStore, *Service, string) {
		repo := newMemRepository()
		repo.seed(1, "Old")
		store := &fakeLogoStore{}
		svc := NewService(repo, store)
		got, err := svc.Update(ctx, 1, validReq(), LogoChange{Add: []byte("first")})
		if err != nil {
			t.Fatal(err)
		}
		store.removed = nil
		return repo, store, svc, logoOf(got)
	}

	t.Run("a logo that cannot be stored", func(t *testing.T) {
		_, store, svc, first := setup()
		store.failErr = ErrUnsupportedMedia
		req := validReq()
		req.Name = "Changed"
		if _, err := svc.Update(ctx, 1, req, LogoChange{Add: []byte("bad")}); !errors.Is(err, ErrUnsupportedMedia) {
			t.Fatalf("err = %v", err)
		}
		got, _ := svc.Get(ctx, 1)
		if got.Name == "Changed" || logoOf(got) != first || len(store.removed) != 0 {
			t.Fatalf("shop = %+v, removed %v: nothing may change", got, store.removed)
		}
	})
	t.Run("the save fails after the logo was stored", func(t *testing.T) {
		repo, store, svc, first := setup()
		repo.failUpdate = true
		if _, err := svc.Update(ctx, 1, validReq(), LogoChange{Add: []byte("second")}); err == nil {
			t.Fatal("want an error")
		}
		newest := store.saved[len(store.saved)-1]
		if len(store.removed) != 1 || store.removed[0] != newest {
			t.Fatalf("removed = %v, want the new file %q and not the old logo", store.removed, newest)
		}
		repo.failUpdate = false
		if got, _ := svc.Get(ctx, 1); logoOf(got) != first {
			t.Fatalf("logo = %q, want the old one %q", logoOf(got), first)
		}
	})
	t.Run("invalid fields never reach the logo store", func(t *testing.T) {
		_, store, svc, _ := setup()
		before := store.next
		req := validReq()
		req.Name = ""
		if _, err := svc.Update(ctx, 1, req, LogoChange{Add: []byte("x")}); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v", err)
		}
		if store.next != before {
			t.Fatal("an image was processed for an invalid request")
		}
	})
	t.Run("a logo and remove_logo together", func(t *testing.T) {
		_, store, svc, _ := setup()
		before := store.next
		if _, err := svc.Update(ctx, 1, validReq(), LogoChange{Add: []byte("x"), Remove: true}); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v", err)
		}
		if store.next != before {
			t.Fatal("an image was processed for a contradictory request")
		}
	})
	t.Run("an unknown shop stores nothing", func(t *testing.T) {
		_, store, svc, _ := setup()
		before := store.next
		if _, err := svc.Update(ctx, 99, validReq(), LogoChange{Add: []byte("x")}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
		if store.next != before {
			t.Fatal("an image was processed for an unknown shop")
		}
	})
}
