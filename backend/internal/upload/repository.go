package upload

import (
	"context"
	"time"
)

// Stored is a file the repository keeps.
type Stored struct {
	ShopID  int64
	Name    string
	ModTime time.Time
}

// Repository stores processed images.
type Repository interface {
	Save(ctx context.Context, shopID int64, name string, data []byte) error
	Delete(ctx context.Context, shopID int64, name string) error
	// List returns every file of every shop, including leftovers of an interrupted save.
	List(ctx context.Context) ([]Stored, error)
}
