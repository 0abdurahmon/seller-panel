package shop

import (
	"context"
	"sync"
)

type memRepository struct {
	mu    sync.Mutex
	shops map[int64]Shop
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
