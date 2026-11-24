package balance

import "context"

// Repository reads balance entries; every method is scoped by shopID.
type Repository interface {
	// Summary sums the shop's entries; a shop with no entries gets zeros.
	Summary(ctx context.Context, shopID int64) (Summary, error)
	// Entries returns one page, newest first, and the total number of entries.
	Entries(ctx context.Context, shopID int64, limit, offset int) ([]Entry, int, error)
}
