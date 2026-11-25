package product

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func validInput() CreateInput {
	return CreateInput{CategoryID: 1, Title: "Phone X", Description: "Good phone", Price: 1_000_000, SKU: "PX-1", Stock: 10}
}

func onePhoto() [][]byte { return [][]byte{img(0)} }

func newService() *Service {
	return NewService(newMemRepository(), newFakePhotos())
}

func TestCreateValidation(t *testing.T) {
	cases := map[string]func(*CreateInput){
		"missing category":    func(in *CreateInput) { in.CategoryID = 0 },
		"blank title":         func(in *CreateInput) { in.Title = "   " },
		"title too long":      func(in *CreateInput) { in.Title = strings.Repeat("a", maxTitleLen+1) },
		"description too big": func(in *CreateInput) { in.Description = strings.Repeat("a", maxDescriptionLen+1) },
		"blank sku":           func(in *CreateInput) { in.SKU = " " },
		"sku too long":        func(in *CreateInput) { in.SKU = strings.Repeat("a", maxSKULen+1) },
		"zero price":          func(in *CreateInput) { in.Price = 0 },
		"negative price":      func(in *CreateInput) { in.Price = -5 },
		"price too large":     func(in *CreateInput) { in.Price = maxPrice + 1 },
		"archived on create":  func(in *CreateInput) { in.Status = StatusArchived },
		"unknown status":      func(in *CreateInput) { in.Status = "published" },
		"negative stock":      func(in *CreateInput) { in.Stock = -1 },
		"stock too large":     func(in *CreateInput) { in.Stock = maxStock + 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			mutate(&in)
			if _, err := newService().Create(context.Background(), 1, in, onePhoto()); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestCreateDefaultsAndTrims(t *testing.T) {
	in := validInput()
	in.Title = "  Phone X  "
	in.SKU = " PX-1 "
	p, err := newService().Create(context.Background(), 7, in, onePhoto())
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Phone X" || p.SKU != "PX-1" {
		t.Fatalf("not trimmed: %q %q", p.Title, p.SKU)
	}
	if p.Status != StatusDraft {
		t.Fatalf("status = %q, want draft", p.Status)
	}
	if p.ShopID != 7 || p.Reserved != 0 || p.Stock != 10 || p.ID == 0 {
		t.Fatalf("unexpected product: %+v", p)
	}
}

func TestSKUUniquePerShop(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	if _, err := svc.Create(ctx, 1, validInput(), onePhoto()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, 1, validInput(), onePhoto()); !errors.Is(err, ErrSKUTaken) {
		t.Fatalf("same shop err = %v, want ErrSKUTaken", err)
	}
	if _, err := svc.Create(ctx, 2, validInput(), onePhoto()); err != nil {
		t.Fatalf("other shop must accept the same sku: %v", err)
	}
}

func TestOtherShopCannotTouchProduct(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	p, _ := svc.Create(ctx, 1, validInput(), onePhoto())

	if _, err := svc.Get(ctx, 2, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Update(ctx, 2, p.ID, UpdateInput{CategoryID: 1, Title: "Hacked", Price: 1, SKU: "H"}, PhotoChange{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Archive(ctx, 2, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("archive err = %v, want ErrNotFound", err)
	}
	got, _ := svc.Get(ctx, 1, p.ID)
	if got.Title != "Phone X" || got.Status != StatusDraft {
		t.Fatalf("product changed by another shop: %+v", got)
	}
}

func TestGetRejectsNonPositiveID(t *testing.T) {
	for _, id := range []int64{0, -1} {
		if _, err := newService().Get(context.Background(), 1, id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("id %d err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestUpdate(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	p, _ := svc.Create(ctx, 1, validInput(), onePhoto())
	other := validInput()
	other.SKU = "OTHER"
	if _, err := svc.Create(ctx, 1, other, onePhoto()); err != nil {
		t.Fatal(err)
	}

	t.Run("changes fields and keeps stock and status", func(t *testing.T) {
		got, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 2, Title: " New ", Description: "d", Price: 2_000_000, SKU: " PX-1 "}, PhotoChange{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Title != "New" || got.CategoryID != 2 || got.Price != 2_000_000 || got.SKU != "PX-1" {
			t.Fatalf("fields not updated: %+v", got)
		}
		if got.Stock != 10 || got.Reserved != 0 || got.Status != StatusDraft {
			t.Fatalf("stock or status changed: %+v", got)
		}
	})
	t.Run("activates", func(t *testing.T) {
		got, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 1, Title: "New", Price: 1, SKU: "PX-1", Status: StatusActive}, PhotoChange{})
		if err != nil || got.Status != StatusActive {
			t.Fatalf("got %+v, err %v", got, err)
		}
	})
	t.Run("sku of another product is taken", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 1, Title: "New", Price: 1, SKU: "OTHER"}, PhotoChange{})
		if !errors.Is(err, ErrSKUTaken) {
			t.Fatalf("err = %v, want ErrSKUTaken", err)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 1, Title: "", Price: 1, SKU: "PX-1"}, PhotoChange{})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
	})
	t.Run("cannot archive through update", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 1, Title: "New", Price: 1, SKU: "PX-1", Status: StatusArchived}, PhotoChange{})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
	})
}

func TestArchive(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	p, _ := svc.Create(ctx, 1, validInput(), onePhoto())

	got, err := svc.Archive(ctx, 1, p.ID)
	if err != nil || got.Status != StatusArchived {
		t.Fatalf("got %+v, err %v", got, err)
	}
	if again, err := svc.Archive(ctx, 1, p.ID); err != nil || again.Status != StatusArchived {
		t.Fatalf("archive must be idempotent: %+v, %v", again, err)
	}
	if _, err := svc.Update(ctx, 1, p.ID, UpdateInput{CategoryID: 1, Title: "Edit", Price: 1, SKU: "PX-1"}, PhotoChange{}); !errors.Is(err, ErrArchived) {
		t.Fatalf("update archived err = %v, want ErrArchived", err)
	}
}

func seedList(t *testing.T) *Service {
	t.Helper()
	svc := newService()
	ctx := context.Background()
	add := func(shop int64, title, sku string, cat int64, status Status) {
		in := CreateInput{CategoryID: cat, Title: title, Price: 1000, SKU: sku, Status: status}
		if _, err := svc.Create(ctx, shop, in, onePhoto()); err != nil {
			t.Fatal(err)
		}
	}
	add(1, "Red Phone", "RP-1", 1, StatusActive)
	add(1, "Blue Phone", "BP-2", 1, StatusDraft)
	add(1, "Kettle", "K-3", 2, StatusActive)
	add(2, "Other shop phone", "OP-1", 1, StatusActive)
	return svc
}

func TestListFilters(t *testing.T) {
	svc := seedList(t)
	ctx := context.Background()
	cases := []struct {
		name  string
		f     ListFilter
		total int
	}{
		{"all of the shop only", ListFilter{}, 3},
		{"by status", ListFilter{Status: StatusDraft}, 1},
		{"by category", ListFilter{CategoryID: 2}, 1},
		{"query on title ignores case", ListFilter{Query: "  PHONE "}, 2},
		{"query on sku", ListFilter{Query: "k-3"}, 1},
		{"no match", ListFilter{Query: "zzz"}, 0},
		{"combined", ListFilter{Query: "phone", Status: StatusActive}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, err := svc.List(ctx, 1, tc.f)
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != tc.total || len(page.Items) != tc.total {
				t.Fatalf("total = %d, items = %d, want %d", page.Total, len(page.Items), tc.total)
			}
			if page.Items == nil {
				t.Fatal("items must be an empty slice, not nil")
			}
		})
	}
}

func TestListRefusesAQueryTheDatabaseCannotStore(t *testing.T) {
	svc := seedList(t)
	for _, q := range []string{"a\x00b", "\xff"} {
		if _, err := svc.List(context.Background(), 1, ListFilter{Query: q}); !errors.Is(err, ErrValidation) {
			t.Errorf("q %q: err = %v, want ErrValidation", q, err)
		}
	}
}

func TestListPagination(t *testing.T) {
	svc := seedList(t)
	ctx := context.Background()

	page, _ := svc.List(ctx, 1, ListFilter{Page: 0, PageSize: 0})
	if page.Page != 1 || page.PageSize != defaultPageSize {
		t.Fatalf("defaults: page %d size %d", page.Page, page.PageSize)
	}
	page, _ = svc.List(ctx, 1, ListFilter{PageSize: 1000})
	if page.PageSize != maxPageSize {
		t.Fatalf("size = %d, want clamp to %d", page.PageSize, maxPageSize)
	}
	first, _ := svc.List(ctx, 1, ListFilter{Page: 1, PageSize: 2})
	second, _ := svc.List(ctx, 1, ListFilter{Page: 2, PageSize: 2})
	if len(first.Items) != 2 || len(second.Items) != 1 || first.Total != 3 {
		t.Fatalf("pages: %d, %d, total %d", len(first.Items), len(second.Items), first.Total)
	}
	beyond, _ := svc.List(ctx, 1, ListFilter{Page: 9, PageSize: 2})
	if len(beyond.Items) != 0 || beyond.Items == nil {
		t.Fatalf("beyond last page must be an empty slice: %+v", beyond.Items)
	}
}

func TestListValidation(t *testing.T) {
	svc := seedList(t)
	for name, f := range map[string]ListFilter{
		"unknown status":    {Status: "published"},
		"negative category": {CategoryID: -1},
		"query too long":    {Query: strings.Repeat("a", maxQueryLen+1)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.List(context.Background(), 1, f); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestLengthLimitsCountCharacters(t *testing.T) {
	cases := map[string]struct {
		limit int
		set   func(in *CreateInput, v string)
	}{
		"title":       {maxTitleLen, func(in *CreateInput, v string) { in.Title = v }},
		"description": {maxDescriptionLen, func(in *CreateInput, v string) { in.Description = v }},
		"sku":         {maxSKULen, func(in *CreateInput, v string) { in.SKU = v }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := newService()
			in := validInput()
			tc.set(&in, strings.Repeat("ж", tc.limit))
			if _, err := svc.Create(context.Background(), 1, in, onePhoto()); err != nil {
				t.Fatalf("%d characters: %v", tc.limit, err)
			}
			tc.set(&in, strings.Repeat("ж", tc.limit+1))
			if _, err := svc.Create(context.Background(), 1, in, onePhoto()); !errors.Is(err, ErrValidation) {
				t.Fatalf("%d characters: err = %v, want ErrValidation", tc.limit+1, err)
			}
		})
	}
	t.Run("q", func(t *testing.T) {
		svc := newService()
		if _, err := svc.List(context.Background(), 1, ListFilter{Query: strings.Repeat("ж", maxQueryLen)}); err != nil {
			t.Fatalf("%d characters: %v", maxQueryLen, err)
		}
		_, err := svc.List(context.Background(), 1, ListFilter{Query: strings.Repeat("ж", maxQueryLen+1)})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("%d characters: err = %v, want ErrValidation", maxQueryLen+1, err)
		}
	})
}

func TestUpdateDoesNotOverwriteAConcurrentChange(t *testing.T) {
	ctx := context.Background()
	up := UpdateInput{CategoryID: 1, Title: "New", Price: 5, SKU: "PX-1", Status: StatusActive}

	t.Run("a product archived meanwhile stays archived", func(t *testing.T) {
		store := newFakePhotos()
		svc := NewService(newMemRepository(), store)
		p, err := svc.Create(ctx, 1, validInput(), onePhoto())
		if err != nil {
			t.Fatal(err)
		}
		store.during = func() {
			if _, err := svc.Archive(ctx, 1, p.ID); err != nil {
				t.Error(err)
			}
		}
		if _, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(1)}}); !errors.Is(err, ErrArchived) {
			t.Fatalf("err = %v, want ErrArchived", err)
		}
		got, err := svc.Get(ctx, 1, p.ID)
		if err != nil || got.Status != StatusArchived || len(store.live()) != 1 {
			t.Fatalf("status = %s, live = %v, err = %v", got.Status, store.live(), err)
		}
	})
	t.Run("a photo added meanwhile stays", func(t *testing.T) {
		store := newFakePhotos()
		svc := NewService(newMemRepository(), store)
		p, err := svc.Create(ctx, 1, validInput(), onePhoto())
		if err != nil {
			t.Fatal(err)
		}
		store.during = func() {
			if _, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(2)}}); err != nil {
				t.Error(err)
			}
		}
		got, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(1)}})
		if err != nil || len(got.Photos) != 3 || len(store.live()) != 3 {
			t.Fatalf("photos = %v, live = %v, err = %v", got.Photos, store.live(), err)
		}
	})
	t.Run("the photo limit counts photos added meanwhile", func(t *testing.T) {
		store := newFakePhotos()
		svc := NewService(newMemRepository(), store)
		p, err := svc.Create(ctx, 1, validInput(), [][]byte{img(1), img(2), img(3), img(4)})
		if err != nil {
			t.Fatal(err)
		}
		store.during = func() {
			if _, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(5)}}); err != nil {
				t.Error(err)
			}
		}
		if _, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Add: [][]byte{img(6)}}); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
		got, err := svc.Get(ctx, 1, p.ID)
		if err != nil || len(got.Photos) != maxPhotos || len(store.live()) != maxPhotos {
			t.Fatalf("photos = %v, live = %v, err = %v", got.Photos, store.live(), err)
		}
	})
}

func TestProductAlwaysHasAPhoto(t *testing.T) {
	ctx := context.Background()
	up := UpdateInput{CategoryID: 1, Title: "New", Price: 5, SKU: "PX-1"}

	t.Run("create without a photo is refused", func(t *testing.T) {
		if _, err := newService().Create(ctx, 1, validInput(), nil); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
	})
	t.Run("update cannot remove the last photo", func(t *testing.T) {
		store := newFakePhotos()
		svc := NewService(newMemRepository(), store)
		p, err := svc.Create(ctx, 1, validInput(), [][]byte{img(1)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Keep: &[]string{}}); !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
		got, err := svc.Get(ctx, 1, p.ID)
		if err != nil || len(got.Photos) != 1 || len(store.removed) != 0 {
			t.Fatalf("photos = %v, removed = %v, err = %v", got.Photos, store.removed, err)
		}
	})
	t.Run("the only photo can be replaced", func(t *testing.T) {
		svc := newService()
		p, err := svc.Create(ctx, 1, validInput(), [][]byte{img(1)})
		if err != nil {
			t.Fatal(err)
		}
		got, err := svc.Update(ctx, 1, p.ID, up, PhotoChange{Keep: &[]string{}, Add: [][]byte{img(2)}})
		if err != nil || len(got.Photos) != 1 || got.Photos[0] == p.Photos[0] {
			t.Fatalf("photos = %v, err = %v", got.Photos, err)
		}
	})
}
