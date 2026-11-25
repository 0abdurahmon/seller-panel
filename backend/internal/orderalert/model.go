package orderalert

// Line is one product of an order, with the title and price it was ordered at.
type Line struct {
	Title    string
	Quantity int
	Price    int64
}
