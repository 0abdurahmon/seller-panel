package order

import (
	"context"
	"math/rand/v2"
)

type fakeBuyer struct {
	name, phone, address string
}

var fakeBuyers = []fakeBuyer{
	{"Aziz Karimov", "+998901112233", "Tashkent, Chilonzor 7, apt 12"},
	{"Dilnoza Yusupova", "+998933334455", "Tashkent, Yunusabad 4, apt 31"},
	{"Jasur Rakhimov", "+998977778899", "Samarkand, Registan 15"},
	{"Madina Ismoilova", "+998909990011", "Bukhara, Mustaqillik 22"},
	{"Sardor Aliyev", "+998941234567", "Namangan, Navoi 3, apt 8"},
	{"Nodira Tursunova", "+998991112200", "Fergana, Mustaqillik 41"},
}

const maxSimulatedItems = 3

func (s *Service) Simulate(ctx context.Context, shopID int64) (Order, error) {
	products, err := s.repo.SimulatableProducts(ctx, shopID)
	if err != nil {
		return Order{}, err
	}
	if len(products) == 0 {
		return Order{}, ErrNoProducts
	}

	lines := 1 + s.intn(min(maxSimulatedItems, len(products)))
	items := make([]NewItem, 0, lines)
	for i := 0; i < lines; i++ {
		j := i + s.intn(len(products)-i)
		products[i], products[j] = products[j], products[i]
		p := products[i]
		items = append(items, NewItem{
			ProductID: p.ID,
			Title:     p.Title,
			Price:     p.Price,
			Quantity:  1 + s.intn(min(maxSimulatedItems, p.Available)),
		})
	}
	buyer := fakeBuyers[s.intn(len(fakeBuyers))]
	return s.repo.Create(ctx, shopID, NewOrder{
		BuyerName:       buyer.name,
		BuyerPhone:      buyer.phone,
		DeliveryAddress: buyer.address,
		Items:           items,
	}, s.now())
}

func randomIntn(n int) int {
	return rand.IntN(n)
}
