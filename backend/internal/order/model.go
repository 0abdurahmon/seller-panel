package order

import "time"

// Status is an order status. Values match the DB enum and docs/DOMAIN.md.
type Status string

const (
	StatusNew             Status = "new"
	StatusAccepted        Status = "accepted"
	StatusPacked          Status = "packed"
	StatusHandedToCourier Status = "handed_to_courier"
	StatusDelivered       Status = "delivered"
	StatusCancelled       Status = "cancelled"
)

// ValidStatus reports whether s is a known status.
func ValidStatus(s Status) bool {
	switch s {
	case StatusNew, StatusAccepted, StatusPacked, StatusHandedToCourier, StatusDelivered, StatusCancelled:
		return true
	}
	return false
}

// allowed is the status machine from docs/DOMAIN.md.
var allowed = map[Status][]Status{
	StatusNew:             {StatusAccepted, StatusCancelled},
	StatusAccepted:        {StatusPacked, StatusCancelled},
	StatusPacked:          {StatusHandedToCourier, StatusCancelled},
	StatusHandedToCourier: {StatusDelivered},
}

// CanTransition reports whether an order may move from one status to another.
func CanTransition(from, to Status) bool {
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return false
}

// holdsReservation reports whether an order in this status has its stock reserved.
func holdsReservation(s Status) bool {
	return s == StatusAccepted || s == StatusPacked || s == StatusHandedToCourier
}

// Item is one line of an order. Title and Price are snapshots taken at order time.
type Item struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Title     string `json:"title"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

// Order is a full order with items.
type Order struct {
	ID              int64          `json:"id"`
	ShopID          int64          `json:"-"`
	Number          string         `json:"number"`
	Status          Status         `json:"status"`
	BuyerName       string         `json:"buyer_name"`
	BuyerPhone      string         `json:"buyer_phone"`
	DeliveryAddress string         `json:"delivery_address"`
	Items           []Item         `json:"items"`
	Total           int64          `json:"total"`
	CancelReason    *string        `json:"cancel_reason"`
	CreatedAt       time.Time      `json:"created_at"`
	StatusChangedAt time.Time      `json:"status_changed_at"`
	History         []HistoryEntry `json:"history"`
}

// HistoryEntry is one status change of an order; FromStatus is nil for the first row.
type HistoryEntry struct {
	FromStatus *Status   `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	Reason     *string   `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// Summary is an order without items, used in lists.
type Summary struct {
	ID        int64     `json:"id"`
	Number    string    `json:"number"`
	Status    Status    `json:"status"`
	BuyerName string    `json:"buyer_name"`
	Total     int64     `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

// Page is one page of results.
type Page[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// TransitionRequest is the body of POST /orders/{id}/transition.
type TransitionRequest struct {
	To     Status `json:"to"`
	Reason string `json:"reason"`
}
