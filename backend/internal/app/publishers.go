package app

import (
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

// fanout sends each order event to every publisher, in order.
type fanout []order.Publisher

func (f fanout) Publish(shopID int64, name string, data any) {
	for _, p := range f {
		p.Publish(shopID, name, data)
	}
}
