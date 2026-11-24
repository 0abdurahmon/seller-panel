package dashboard

import (
	"context"
	"sort"
	"time"
)

type memItem struct {
	productID int64
	quantity  int64
	price     int64
}

type memOrder struct {
	shopID    int64
	status    string
	total     int64
	createdAt time.Time
	changedAt time.Time
	items     []memItem
}

// memRepository applies the same rules as the SQL: windows are half open, revenue counts
// delivered orders by the time they were delivered, and the top list is ranked and cut at 5.
type memRepository struct {
	orders []memOrder
	titles map[int64]string
	err    error
	last   Window
}

func newMemRepository() *memRepository {
	return &memRepository{titles: map[int64]string{}}
}

func (m *memRepository) add(o memOrder) {
	m.orders = append(m.orders, o)
}

func in(t, from, to time.Time) bool {
	return !t.Before(from) && t.Before(to)
}

func (m *memRepository) Summary(_ context.Context, shopID int64, w Window) (Summary, error) {
	m.last = w
	if m.err != nil {
		return Summary{}, m.err
	}
	var s Summary
	type sold struct{ quantity, revenue int64 }
	perProduct := map[int64]*sold{}
	for _, o := range m.orders {
		if o.shopID != shopID {
			continue
		}
		if in(o.createdAt, w.TodayStart, w.TodayEnd) {
			s.TodayOrders++
		}
		switch o.status {
		case "new", "accepted", "packed":
			s.WaitingForAction++
		case "delivered":
			if in(o.changedAt, w.From7D, w.TodayEnd) {
				s.Revenue7D += o.total
			}
			if in(o.changedAt, w.From30D, w.TodayEnd) {
				s.Revenue30D += o.total
				for _, it := range o.items {
					p := perProduct[it.productID]
					if p == nil {
						p = &sold{}
						perProduct[it.productID] = p
					}
					p.quantity += it.quantity
					p.revenue += it.price * it.quantity
				}
			}
		}
	}
	for id, p := range perProduct {
		s.TopProducts = append(s.TopProducts, TopProduct{ProductID: id, Title: m.titles[id], QuantitySold: p.quantity, Revenue: p.revenue})
	}
	sort.Slice(s.TopProducts, func(i, j int) bool {
		a, b := s.TopProducts[i], s.TopProducts[j]
		if a.QuantitySold != b.QuantitySold {
			return a.QuantitySold > b.QuantitySold
		}
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		return a.ProductID < b.ProductID
	})
	if len(s.TopProducts) > 5 {
		s.TopProducts = s.TopProducts[:5]
	}
	return s, nil
}
