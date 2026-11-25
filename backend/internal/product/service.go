package product

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxTitleLen       = 200
	maxDescriptionLen = 5000
	maxSKULen         = 64
	maxQueryLen       = 100
	maxPrice          = 1_000_000_000_000
	maxStock          = 1_000_000
	defaultPageSize   = 20
	maxPageSize       = 100
	maxPage           = 1_000_000
	maxPhotos         = 5
	photoRequired     = "a product needs at least one photo"
)

type Service struct {
	repo   Repository
	photos PhotoStore
}

func NewService(repo Repository, photos PhotoStore) *Service {
	return &Service{repo: repo, photos: photos}
}

// Create stores the product and its new photos together: when any step fails, no product and no file is left.
func (s *Service) Create(ctx context.Context, shopID int64, in CreateInput, add [][]byte) (Product, error) {
	p := Product{
		ShopID:      shopID,
		CategoryID:  in.CategoryID,
		Title:       in.Title,
		Description: in.Description,
		Price:       in.Price,
		SKU:         in.SKU,
		Stock:       in.Stock,
		Status:      in.Status,
	}
	if p.Status == "" {
		p.Status = StatusDraft
	}
	if err := validate(&p); err != nil {
		return Product{}, err
	}
	if p.Stock < 0 || p.Stock > maxStock {
		return Product{}, invalid("stock must be between 0 and %d", maxStock)
	}
	if len(add) == 0 {
		return Product{}, invalid(photoRequired)
	}
	if len(add) > maxPhotos {
		return Product{}, invalid("a product has at most %d photos", maxPhotos)
	}
	saved, err := s.savePhotos(ctx, shopID, add)
	if err != nil {
		return Product{}, err
	}
	p.Photos = saved
	created, err := s.repo.Create(ctx, shopID, p)
	if err != nil {
		s.photos.Remove(ctx, shopID, saved)
		return Product{}, err
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, shopID, id int64) (Product, error) {
	if id <= 0 {
		return Product{}, ErrNotFound
	}
	return s.repo.Get(ctx, shopID, id)
}

// Update changes the product and its photos together. Photos that are no longer listed are deleted after the
// product is saved; new files are deleted again when the save fails.
func (s *Service) Update(ctx context.Context, shopID, id int64, in UpdateInput, pc PhotoChange) (Product, error) {
	p, err := s.Get(ctx, shopID, id)
	if err != nil {
		return Product{}, err
	}
	// A refused request must not store files, so it is checked before they are saved and again under the lock.
	if _, err := edited(p, in, pc.Keep, make([]string, len(pc.Add))); err != nil {
		return Product{}, err
	}
	saved, err := s.savePhotos(ctx, shopID, pc.Add)
	if err != nil {
		return Product{}, err
	}
	var previous []string
	updated, err := s.repo.Update(ctx, shopID, id, func(current Product) (Product, error) {
		previous = current.Photos
		return edited(current, in, pc.Keep, saved)
	})
	if err != nil {
		s.photos.Remove(ctx, shopID, saved)
		return Product{}, err
	}
	s.photos.Remove(ctx, shopID, dropped(previous, updated.Photos))
	return updated, nil
}

// edited returns p with the request applied, or why the request is refused for p.
func edited(p Product, in UpdateInput, keep *[]string, added []string) (Product, error) {
	if p.Status == StatusArchived {
		return Product{}, ErrArchived
	}
	p.CategoryID = in.CategoryID
	p.Title = in.Title
	p.Description = in.Description
	p.Price = in.Price
	p.SKU = in.SKU
	if in.Status != "" {
		p.Status = in.Status
	}
	if err := validate(&p); err != nil {
		return Product{}, err
	}
	kept, err := keptPhotos(p.Photos, keep)
	if err != nil {
		return Product{}, err
	}
	if len(kept)+len(added) == 0 {
		return Product{}, invalid(photoRequired)
	}
	if len(kept)+len(added) > maxPhotos {
		return Product{}, invalid("a product has at most %d photos, %d kept and %d new", maxPhotos, len(kept), len(added))
	}
	p.Photos = append(append([]string{}, kept...), added...)
	return p, nil
}

func (s *Service) Archive(ctx context.Context, shopID, id int64) (Product, error) {
	p, err := s.Get(ctx, shopID, id)
	if err != nil {
		return Product{}, err
	}
	if p.Status == StatusArchived {
		return p, nil
	}
	return s.repo.Update(ctx, shopID, id, func(current Product) (Product, error) {
		current.Status = StatusArchived
		return current, nil
	})
}

func (s *Service) List(ctx context.Context, shopID int64, f ListFilter) (Page, error) {
	f.Query = strings.TrimSpace(f.Query)
	if !utf8.ValidString(f.Query) || strings.ContainsRune(f.Query, 0) {
		return Page{}, invalid("q must be valid text")
	}
	if utf8.RuneCountInString(f.Query) > maxQueryLen {
		return Page{}, invalid("q must be at most %d characters", maxQueryLen)
	}
	if f.CategoryID < 0 {
		return Page{}, invalid("category must not be negative")
	}
	if f.Status != "" && !f.Status.valid() {
		return Page{}, invalid("status must be draft, active or archived")
	}
	if f.Page > maxPage {
		return Page{}, invalid("page must be at most %d", maxPage)
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = defaultPageSize
	}
	if f.PageSize > maxPageSize {
		f.PageSize = maxPageSize
	}

	items, total, err := s.repo.List(ctx, shopID, f)
	if err != nil {
		return Page{}, err
	}
	if items == nil {
		items = []Product{}
	}
	return Page{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

// savePhotos stores the new images in order. When one fails, the ones stored before it are removed again.
func (s *Service) savePhotos(ctx context.Context, shopID int64, add [][]byte) ([]string, error) {
	saved := make([]string, 0, len(add))
	for i, data := range add {
		url, err := s.photos.Save(ctx, shopID, bytes.NewReader(data))
		if err != nil {
			s.photos.Remove(ctx, shopID, saved)
			return nil, fmt.Errorf("photos[%d]: %w", i, err)
		}
		saved = append(saved, url)
	}
	return saved, nil
}

// keptPhotos returns the photos that stay: all current ones when keep is nil, otherwise the listed ones,
// each of which must be a current photo of the product and listed once.
func keptPhotos(current []string, keep *[]string) ([]string, error) {
	if keep == nil {
		return current, nil
	}
	have := make(map[string]bool, len(current))
	for _, u := range current {
		have[u] = true
	}
	out := make([]string, 0, len(*keep))
	seen := make(map[string]bool, len(*keep))
	for i, u := range *keep {
		u = strings.TrimSpace(u)
		switch {
		case !have[u]:
			return nil, invalid("keep_photos[%d] is not a photo of this product", i)
		case seen[u]:
			return nil, invalid("keep_photos[%d] is listed twice", i)
		}
		seen[u] = true
		out = append(out, u)
	}
	return out, nil
}

func dropped(current, kept []string) []string {
	stay := make(map[string]bool, len(kept))
	for _, u := range kept {
		stay[u] = true
	}
	var gone []string
	for _, u := range current {
		if !stay[u] {
			gone = append(gone, u)
		}
	}
	return gone
}

func validate(p *Product) error {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	p.SKU = strings.TrimSpace(p.SKU)

	switch {
	case p.CategoryID <= 0:
		return invalid("category_id is required")
	case p.Title == "" || utf8.RuneCountInString(p.Title) > maxTitleLen:
		return invalid("title must be 1-%d characters", maxTitleLen)
	case utf8.RuneCountInString(p.Description) > maxDescriptionLen:
		return invalid("description must be at most %d characters", maxDescriptionLen)
	case p.SKU == "" || utf8.RuneCountInString(p.SKU) > maxSKULen:
		return invalid("sku must be 1-%d characters", maxSKULen)
	case p.Price <= 0 || p.Price > maxPrice:
		return invalid("price must be a positive integer up to %d", int64(maxPrice))
	case p.Status != StatusDraft && p.Status != StatusActive:
		return invalid("status must be draft or active")
	}
	return nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}
