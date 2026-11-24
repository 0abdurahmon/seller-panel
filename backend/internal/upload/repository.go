package upload

import (
	"context"
	"io"
)

// Repository stores processed images.
type Repository interface {
	Save(ctx context.Context, shopID int64, name string, data []byte) error
	Open(ctx context.Context, shopID int64, name string) (io.ReadCloser, error)
	Delete(ctx context.Context, shopID int64, name string) error
}
