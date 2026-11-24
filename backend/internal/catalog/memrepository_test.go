package catalog

import "context"

type memRepository struct {
	items []Category
	err   error
}

func (r *memRepository) List(_ context.Context) ([]Category, error) {
	return r.items, r.err
}
