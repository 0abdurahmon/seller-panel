package upload

import "context"

// Repository stores processed images.
type Repository interface {
	Save(ctx context.Context, shopID int64, name string, data []byte) error
	Delete(ctx context.Context, shopID int64, name string) error
}
