// Package balance shows the money a shop has earned: totals and history (see D7).
package balance

import "time"

// EntryType is the kind of money movement.
type EntryType string

const (
	EntrySale       EntryType = "sale"
	EntryCommission EntryType = "commission"
	EntryPayout     EntryType = "payout"
)

// Summary is the shop's balance in so'm.
type Summary struct {
	Available  int64 `json:"available"`
	Earned     int64 `json:"earned"`
	Commission int64 `json:"commission"`
}

// Entry is one money movement; OrderID is nil for payouts.
type Entry struct {
	ID        int64     `json:"id"`
	OrderID   *int64    `json:"order_id"`
	Type      EntryType `json:"type"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// Page is one page of entries, newest first; Items is never nil.
type Page struct {
	Items    []Entry `json:"items"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	Total    int     `json:"total"`
}
