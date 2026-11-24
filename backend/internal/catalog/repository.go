package catalog

import "context"

type Repository interface {
	List(ctx context.Context) ([]Category, error)
}
