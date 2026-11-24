package product

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type memRepository struct {
	next  int64
	items map[int64]Product
}

func newMemRepository() *memRepository {
	return &memRepository{items: make(map[int64]Product)}
}

func (r *memRepository) skuTaken(shopID int64, sku string, exceptID int64) bool {
	for _, p := range r.items {
		if p.ShopID == shopID && p.SKU == sku && p.ID != exceptID {
			return true
		}
	}
	return false
}

func (r *memRepository) Create(_ context.Context, shopID int64, p Product) (Product, error) {
	if r.skuTaken(shopID, p.SKU, 0) {
		return Product{}, ErrSKUTaken
	}
	r.next++
	p.ID = r.next
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	r.items[p.ID] = p
	return p, nil
}

func (r *memRepository) Get(_ context.Context, shopID, id int64) (Product, error) {
	p, ok := r.items[id]
	if !ok || p.ShopID != shopID {
		return Product{}, ErrNotFound
	}
	return p, nil
}

func (r *memRepository) Update(_ context.Context, shopID int64, p Product) (Product, error) {
	cur, ok := r.items[p.ID]
	if !ok || cur.ShopID != shopID {
		return Product{}, ErrNotFound
	}
	if r.skuTaken(shopID, p.SKU, p.ID) {
		return Product{}, ErrSKUTaken
	}
	p.UpdatedAt = time.Now()
	r.items[p.ID] = p
	return p, nil
}

func (r *memRepository) List(_ context.Context, shopID int64, f ListFilter) ([]Product, int, error) {
	var all []Product
	q := strings.ToLower(f.Query)
	for _, p := range r.items {
		switch {
		case p.ShopID != shopID:
		case f.Status != "" && p.Status != f.Status:
		case f.CategoryID != 0 && p.CategoryID != f.CategoryID:
		case q != "" && !strings.Contains(strings.ToLower(p.Title), q) && !strings.Contains(strings.ToLower(p.SKU), q):
		default:
			all = append(all, p)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	total := len(all)
	start := (f.Page - 1) * f.PageSize
	if start > total {
		start = total
	}
	end := start + f.PageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

// fakePhotoStore stores nothing real: Save hands out numbered URLs, Remove records what was deleted.
type fakePhotoStore struct {
	mu      sync.Mutex
	next    int
	saved   []string
	removed []string
	failAt  int
	failErr error
}

func newFakePhotos() *fakePhotoStore { return &fakePhotoStore{} }

func (f *fakePhotoStore) Save(_ context.Context, shopID int64, image io.Reader) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, _ := io.ReadAll(image)
	f.next++
	if f.failAt != 0 && f.next == f.failAt {
		return "", f.failErr
	}
	url := fmt.Sprintf("/api/uploads/%d/%032d.webp", shopID, f.next)
	if len(data) > 0 {
		f.saved = append(f.saved, url)
	}
	return url, nil
}

func (f *fakePhotoStore) Remove(_ context.Context, _ int64, urls []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, urls...)
}

func (f *fakePhotoStore) live() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	gone := map[string]bool{}
	for _, u := range f.removed {
		gone[u] = true
	}
	var out []string
	for _, u := range f.saved {
		if !gone[u] {
			out = append(out, u)
		}
	}
	return out
}
