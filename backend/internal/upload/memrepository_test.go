package upload

import (
	"context"
	"fmt"
	"sync"
)

type memRepository struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMemRepository() *memRepository {
	return &memRepository{files: map[string][]byte{}}
}

func key(shopID int64, name string) string { return fmt.Sprintf("%d/%s", shopID, name) }

func (m *memRepository) Save(_ context.Context, shopID int64, name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key(shopID, name)] = data
	return nil
}

func (m *memRepository) Delete(_ context.Context, shopID int64, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := key(shopID, name)
	if _, ok := m.files[k]; !ok {
		return ErrNotFound
	}
	delete(m.files, k)
	return nil
}

func (m *memRepository) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}
