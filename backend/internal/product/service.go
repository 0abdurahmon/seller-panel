package product

import (
	"context"
	"fmt"
	"strings"
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
	maxPhotos         = 5
	maxPhotoURLLen    = 2048
)

type Service struct {
	repo   Repository
	photos PhotoStore
}

func NewService(repo Repository, photos PhotoStore) *Service {
	return &Service{repo: repo, photos: photos}
}

func (s *Service) Create(ctx context.Context, shopID int64, in CreateInput) (Product, error) {
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
	photos, err := s.checkPhotos(ctx, shopID, in.Photos)
	if err != nil {
		return Product{}, err
	}
	p.Photos = photos
	return s.repo.Create(ctx, shopID, p)
}

func (s *Service) Get(ctx context.Context, shopID, id int64) (Product, error) {
	if id <= 0 {
		return Product{}, ErrNotFound
	}
	return s.repo.Get(ctx, shopID, id)
}

func (s *Service) Update(ctx context.Context, shopID, id int64, in UpdateInput) (Product, error) {
	p, err := s.Get(ctx, shopID, id)
	if err != nil {
		return Product{}, err
	}
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
	if in.Photos != nil {
		if p.Photos, err = s.checkPhotos(ctx, shopID, in.Photos); err != nil {
			return Product{}, err
		}
	}
	return s.repo.Update(ctx, shopID, p)
}

func (s *Service) Archive(ctx context.Context, shopID, id int64) (Product, error) {
	p, err := s.Get(ctx, shopID, id)
	if err != nil {
		return Product{}, err
	}
	if p.Status == StatusArchived {
		return p, nil
	}
	p.Status = StatusArchived
	return s.repo.Update(ctx, shopID, p)
}

func (s *Service) List(ctx context.Context, shopID int64, f ListFilter) (Page, error) {
	f.Query = strings.TrimSpace(f.Query)
	if len(f.Query) > maxQueryLen {
		return Page{}, invalid("q must be at most %d characters", maxQueryLen)
	}
	if f.CategoryID < 0 {
		return Page{}, invalid("category must not be negative")
	}
	if f.Status != "" && !f.Status.valid() {
		return Page{}, invalid("status must be draft, active or archived")
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

// checkPhotos trims and checks the URLs: at most maxPhotos, no repeats, and each one an upload of this shop.
func (s *Service) checkPhotos(ctx context.Context, shopID int64, urls []string) ([]string, error) {
	if len(urls) > maxPhotos {
		return nil, invalid("photos must have at most %d items", maxPhotos)
	}
	out := make([]string, 0, len(urls))
	seen := make(map[string]bool, len(urls))
	for i, u := range urls {
		u = strings.TrimSpace(u)
		switch {
		case u == "" || len(u) > maxPhotoURLLen:
			return nil, invalid("photos[%d] must be a URL returned by POST /uploads", i)
		case seen[u]:
			return nil, invalid("photos[%d] is listed twice", i)
		}
		ok, err := s.photos.Owns(ctx, shopID, u)
		if err != nil {
			return nil, fmt.Errorf("check photo: %w", err)
		}
		if !ok {
			return nil, invalid("photos[%d] is not an image uploaded by your shop", i)
		}
		seen[u] = true
		out = append(out, u)
	}
	return out, nil
}

func validate(p *Product) error {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	p.SKU = strings.TrimSpace(p.SKU)

	switch {
	case p.CategoryID <= 0:
		return invalid("category_id is required")
	case p.Title == "" || len(p.Title) > maxTitleLen:
		return invalid("title must be 1-%d characters", maxTitleLen)
	case len(p.Description) > maxDescriptionLen:
		return invalid("description must be at most %d characters", maxDescriptionLen)
	case p.SKU == "" || len(p.SKU) > maxSKULen:
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
