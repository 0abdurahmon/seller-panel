package product

import (
	"context"
	"io"
	"time"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

func (s Status) valid() bool {
	return s == StatusDraft || s == StatusActive || s == StatusArchived
}

type Product struct {
	ID          int64     `json:"id"`
	ShopID      int64     `json:"shop_id"`
	CategoryID  int64     `json:"category_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int64     `json:"price"`
	SKU         string    `json:"sku"`
	Stock       int       `json:"stock"`
	Reserved    int       `json:"reserved"`
	Status      Status    `json:"status"`
	Photos      []string  `json:"photos"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateInput struct {
	CategoryID  int64  `json:"category_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	SKU         string `json:"sku"`
	Stock       int    `json:"stock"`
	// Status is not part of the request body: a product is created as a draft, the seed sets it directly.
	Status Status `json:"-"`
}

type UpdateInput struct {
	CategoryID  int64  `json:"category_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	SKU         string `json:"sku"`
	Status      Status `json:"status"`
}

type ListFilter struct {
	Query      string
	CategoryID int64
	Status     Status
	Page       int
	PageSize   int
}

type Page struct {
	Items    []Product `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

// PhotoStore turns image bytes into stored files. Save converts and stores one image and returns its URL;
// Remove deletes stored files and never fails the caller (a file that stays behind is only wasted space).
type PhotoStore interface {
	Save(ctx context.Context, shopID int64, image io.Reader) (url string, err error)
	Remove(ctx context.Context, shopID int64, urls []string)
}

// PhotoChange describes what a request does to the photos of a product.
// Keep nil means "keep every current photo"; otherwise exactly the listed current URLs stay, in that order.
// Add holds new images, appended after the kept ones.
type PhotoChange struct {
	Keep *[]string
	Add  [][]byte
}

type Repository interface {
	Create(ctx context.Context, shopID int64, p Product) (Product, error)
	Get(ctx context.Context, shopID, id int64) (Product, error)
	Update(ctx context.Context, shopID int64, p Product) (Product, error)
	List(ctx context.Context, shopID int64, f ListFilter) ([]Product, int, error)
}
