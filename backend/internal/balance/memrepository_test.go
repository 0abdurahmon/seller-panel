package balance

import (
	"context"
	"sort"
	"time"
)

type memEntry struct {
	shopID int64
	entry  Entry
}

type memRepository struct {
	next    int64
	entries []memEntry
}

func newMemRepository() *memRepository {
	return &memRepository{}
}

// add appends an entry the way the order feature would; it returns the new id.
func (r *memRepository) add(shopID int64, orderID *int64, typ EntryType, amount int64, at time.Time) int64 {
	r.next++
	r.entries = append(r.entries, memEntry{shopID: shopID, entry: Entry{
		ID: r.next, OrderID: orderID, Type: typ, Amount: amount, CreatedAt: at,
	}})
	return r.next
}

func (r *memRepository) Summary(_ context.Context, shopID int64) (Summary, error) {
	var s Summary
	for _, e := range r.entries {
		if e.shopID != shopID {
			continue
		}
		s.Available += e.entry.Amount
		switch e.entry.Type {
		case EntrySale:
			s.Earned += e.entry.Amount
		case EntryCommission:
			s.Commission += e.entry.Amount
		}
	}
	return s, nil
}

func (r *memRepository) Entries(_ context.Context, shopID int64, limit, offset int) ([]Entry, int, error) {
	var own []Entry
	for _, e := range r.entries {
		if e.shopID == shopID {
			own = append(own, e.entry)
		}
	}
	sort.Slice(own, func(i, j int) bool {
		if !own[i].CreatedAt.Equal(own[j].CreatedAt) {
			return own[i].CreatedAt.After(own[j].CreatedAt)
		}
		return own[i].ID > own[j].ID
	})
	total := len(own)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return own[offset:end], total, nil
}
