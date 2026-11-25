package shop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

type memRepository struct {
	mu         sync.Mutex
	shops      map[int64]Shop
	failUpdate bool
}

func newMemRepository() *memRepository {
	return &memRepository{shops: make(map[int64]Shop)}
}

func (m *memRepository) seed(id int64, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shops[id] = Shop{ID: id, Name: name}
}

func (m *memRepository) Get(_ context.Context, shopID int64) (Shop, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.shops[shopID]
	if !ok {
		return Shop{}, ErrNotFound
	}
	return s, nil
}

func (m *memRepository) Update(_ context.Context, shopID int64, p Profile) (Shop, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failUpdate {
		return Shop{}, errors.New("database is down")
	}
	if _, ok := m.shops[shopID]; !ok {
		return Shop{}, ErrNotFound
	}
	s := Shop{
		ID:          shopID,
		Name:        p.Name,
		Description: p.Description,
	}
	if p.LogoURL != "" {
		s.LogoURL = &p.LogoURL
	}
	if p.ContactPhone != "" {
		s.ContactPhone = &p.ContactPhone
	}
	if p.PickupAddress != "" {
		s.PickupAddress = &p.PickupAddress
	}
	m.shops[shopID] = s
	return s, nil
}

// fakeLogoStore stores nothing real: Save hands out numbered URLs, Remove records what was deleted.
type fakeLogoStore struct {
	mu      sync.Mutex
	next    int
	saved   []string
	removed []string
	failErr error
}

func (f *fakeLogoStore) Save(_ context.Context, shopID int64, image io.Reader) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failErr != nil {
		return "", f.failErr
	}
	if _, err := io.Copy(io.Discard, image); err != nil {
		return "", err
	}
	f.next++
	url := fmt.Sprintf("/api/uploads/%d/%032d.webp", shopID, f.next)
	f.saved = append(f.saved, url)
	return url, nil
}

func (f *fakeLogoStore) Remove(_ context.Context, _ int64, urls []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, urls...)
}
