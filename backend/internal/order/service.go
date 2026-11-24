package order

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxPage         = 1_000_000
)

// Service holds the order rules: the status machine, stock effects and the
// commission on delivery.
type Service struct {
	repo              Repository
	commissionPercent int64
	now               func() time.Time
	intn              func(n int) int
}

// NewService builds the order service. commissionPercent is 0–100.
func NewService(repo Repository, commissionPercent int) (*Service, error) {
	if commissionPercent < 0 || commissionPercent > 100 {
		return nil, fmt.Errorf("commission percent must be between 0 and 100")
	}
	return &Service{repo: repo, commissionPercent: int64(commissionPercent), now: time.Now, intn: randomIntn}, nil
}

// List returns one page of the shop's orders, newest first. Paging is clamped like product lists.
func (s *Service) List(ctx context.Context, shopID int64, status *Status, page, pageSize int) (Page[Summary], error) {
	if status != nil && !ValidStatus(*status) {
		return Page[Summary]{}, fmt.Errorf("%w: unknown status", ErrValidation)
	}
	if page > maxPage {
		return Page[Summary]{}, fmt.Errorf("%w: page must be at most %d", ErrValidation, maxPage)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	res, err := s.repo.List(ctx, shopID, Filter{Status: status, Page: page, PageSize: pageSize})
	if err != nil {
		return Page[Summary]{}, err
	}
	if res.Items == nil {
		res.Items = []Summary{}
	}
	return res, nil
}

// Get returns one order with its items and status history.
func (s *Service) Get(ctx context.Context, shopID, orderID int64) (Order, error) {
	o, err := s.repo.Get(ctx, shopID, orderID)
	if err != nil {
		return Order{}, err
	}
	return withLists(o), nil
}

func withLists(o Order) Order {
	if o.Items == nil {
		o.Items = []Item{}
	}
	if o.History == nil {
		o.History = []HistoryEntry{}
	}
	return o
}

// Transition moves an order to a new status and applies the side effects in
// one transaction. Any error leaves the order and stock unchanged.
func (s *Service) Transition(ctx context.Context, shopID, orderID int64, req TransitionRequest) (Order, error) {
	if !ValidStatus(req.To) {
		return Order{}, fmt.Errorf("%w: unknown status", ErrValidation)
	}
	reason := strings.TrimSpace(req.Reason)
	if req.To == StatusCancelled && reason == "" {
		return Order{}, fmt.Errorf("%w: reason is required to cancel", ErrValidation)
	}

	var result Order
	err := s.repo.InTx(ctx, func(tx TxRepository) error {
		o, err := tx.LockOrder(ctx, shopID, orderID)
		if err != nil {
			return err
		}
		from := o.Status
		if !CanTransition(from, req.To) {
			return ErrInvalidTransition
		}

		switch req.To {
		case StatusAccepted:
			for _, it := range o.Items {
				if err := tx.Reserve(ctx, shopID, it.ProductID, it.Quantity); err != nil {
					return err
				}
			}
		case StatusCancelled:
			if holdsReservation(from) {
				for _, it := range o.Items {
					if err := tx.Release(ctx, shopID, it.ProductID, it.Quantity); err != nil {
						return err
					}
				}
			}
		case StatusDelivered:
			for _, it := range o.Items {
				if err := tx.Deliver(ctx, shopID, it.ProductID, it.Quantity); err != nil {
					return err
				}
			}
			commission := o.Total * s.commissionPercent / 100
			entries := []BalanceEntry{
				{ShopID: shopID, OrderID: orderID, Type: EntrySale, Amount: o.Total},
				{ShopID: shopID, OrderID: orderID, Type: EntryCommission, Amount: -commission},
			}
			if err := tx.AddBalance(ctx, entries); err != nil {
				return err
			}
		}

		now := s.now()
		var cancelReason *string
		if req.To == StatusCancelled {
			cancelReason = &reason
		}
		if err := tx.SetStatus(ctx, shopID, orderID, req.To, cancelReason, now); err != nil {
			return err
		}
		o.History = append(o.History, HistoryEntry{FromStatus: &from, ToStatus: req.To, Reason: cancelReason, CreatedAt: now})
		o.Status = req.To
		o.CancelReason = cancelReason
		o.StatusChangedAt = now
		result = withLists(o)
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	return result, nil
}
