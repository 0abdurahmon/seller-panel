package product

import (
	"context"
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
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p Product) Available() int {
	return p.Stock - p.Reserved
}

type CreateInput struct {
	CategoryID  int64  `json:"category_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	SKU         string `json:"sku"`
	Stock       int    `json:"stock"`
	Status      Status `json:"status"`
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

type Repository interface {
	Create(ctx context.Context, shopID int64, p Product) (Product, error)
	Get(ctx context.Context, shopID, id int64) (Product, error)
	Update(ctx context.Context, shopID int64, p Product) (Product, error)
	List(ctx context.Context, shopID int64, f ListFilter) ([]Product, int, error)
}
