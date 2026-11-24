package upload

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type memRepository struct {
	mu    sync.Mutex
	files map[string][]byte
	mod   map[string]time.Time
}

func newMemRepository() *memRepository {
	return &memRepository{files: map[string][]byte{}, mod: map[string]time.Time{}}
}

func key(shopID int64, name string) string { return fmt.Sprintf("%d/%s", shopID, name) }

func (m *memRepository) Save(_ context.Context, shopID int64, name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key(shopID, name)] = data
	m.mod[key(shopID, name)] = time.Now()
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
	delete(m.mod, k)
	return nil
}

func (m *memRepository) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

func (m *memRepository) List(context.Context) ([]Stored, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Stored
	for k, mod := range m.mod {
		var shopID int64
		var name string
		fmt.Sscanf(k, "%d/%s", &shopID, &name)
		out = append(out, Stored{ShopID: shopID, Name: name, ModTime: mod})
	}
	return out, nil
}

// put stores a file with a given age, for the sweep tests.
func (m *memRepository) put(shopID int64, name string, age time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key(shopID, name)] = []byte("x")
	m.mod[key(shopID, name)] = time.Now().Add(-age)
}
