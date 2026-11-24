package seed

import "github.com/0abdurahmon/seller-panel/backend/internal/order"

// Demo login printed by the seed command and documented in the README.
const (
	DefaultEmail    = "demo@example.com"
	DefaultPassword = "demo-pass-1234"
	defaultShopName = "Tashkent Home & Tech"
)

// initialStock is high enough for every planned order; the final level is set at the end.
const initialStock = 100

type productSpec struct {
	sku         string
	category    string
	title       string
	description string
	price       int64
	// finalAvailable is what the seller sees at the end; used for active products only.
	finalAvailable int
	status         string
	stock          int
}

// products are 13 active, 1 draft and 1 archived. Prices are in so'm.
var products = []productSpec{
	{sku: "PH-A55", category: "Phones", title: "Samsung Galaxy A55 128 GB", description: "6.6 inch AMOLED, 8 GB RAM, 50 MP camera", price: 4_200_000, finalAvailable: 14},
	{sku: "PH-RN13", category: "Phones", title: "Xiaomi Redmi Note 13", description: "6.67 inch AMOLED, 8 GB RAM, 108 MP camera", price: 2_800_000, finalAvailable: 22},
	{sku: "PH-IP13", category: "Phones", title: "Apple iPhone 13 128 GB", description: "A15 Bionic, dual camera, Face ID", price: 6_900_000, finalAvailable: 8},
	{sku: "LT-IP3", category: "Laptops", title: "Lenovo IdeaPad 3 15 inch", description: "Ryzen 5, 8 GB RAM, 512 GB SSD", price: 6_300_000, finalAvailable: 6},
	{sku: "HP-RB5", category: "Headphones", title: "Xiaomi Redmi Buds 5", description: "Wireless earbuds with noise cancellation", price: 350_000, finalAvailable: 4},
	{sku: "HP-AP2", category: "Headphones", title: "Apple AirPods 2", description: "Wireless earbuds with a charging case", price: 1_650_000, finalAvailable: 3},
	{sku: "MN-TS1", category: "Men", title: "Men's cotton T-shirt", description: "100% cotton, regular fit", price: 89_000, finalAvailable: 120},
	{sku: "MN-JK1", category: "Men", title: "Men's winter jacket", description: "Water resistant, warm lining", price: 560_000, finalAvailable: 18},
	{sku: "WM-DR1", category: "Women", title: "Women's summer dress", description: "Light fabric, knee length", price: 249_000, finalAvailable: 35},
	{sku: "KD-SN1", category: "Kids", title: "Kids sneakers", description: "Breathable mesh, rubber sole", price: 199_000, finalAvailable: 0},
	{sku: "KT-PAN28", category: "Kitchen", title: "Non-stick frying pan 28 cm", description: "Induction ready", price: 149_000, finalAvailable: 40},
	{sku: "KT-KTL17", category: "Kitchen", title: "Electric kettle 1.7 L", description: "Stainless steel, auto shut-off", price: 189_000, finalAvailable: 27},
	{sku: "FR-CH1", category: "Furniture", title: "Office chair with lumbar support", description: "Mesh back, adjustable height", price: 890_000, finalAvailable: 9},
	{sku: "DC-CLK1", category: "Decor", title: "Wall clock 30 cm", description: "Silent movement", price: 95_000, status: "draft", stock: 20},
	{sku: "PH-A34", category: "Phones", title: "Samsung Galaxy A34 (old model)", description: "Replaced by the A55", price: 3_100_000, status: "archived", stock: 5},
}

type buyer struct {
	name, phone, address string
}

var buyers = []buyer{
	{"Aziz Karimov", "+998901112233", "Tashkent, Chilonzor 7, apt 12"},
	{"Dilnoza Yusupova", "+998933334455", "Tashkent, Yunusabad 4, apt 31"},
	{"Jasur Rakhimov", "+998977778899", "Samarkand, Registan 15"},
	{"Madina Ismoilova", "+998909990011", "Bukhara, Mustaqillik 22"},
	{"Sardor Aliyev", "+998941234567", "Namangan, Navoi 3, apt 8"},
	{"Nodira Tursunova", "+998991112200", "Fergana, Mustaqillik 41"},
	{"Bobur Hasanov", "+998935556677", "Andijan, Bobur 18"},
	{"Zarina Abdullaeva", "+998907778800", "Tashkent, Mirzo Ulugbek 9, apt 54"},
}

type line struct {
	sku string
	qty int
}

// plannedOrder is one demo order. Past orders are placed on daysAgo at the given hour of the business
// day; orders of today are placed at the given fraction of the day that has passed.
type plannedOrder struct {
	daysAgo int
	hour    int
	today   float64
	final   order.Status
	// cancelFrom is the status a cancelled order was in when it was cancelled.
	cancelFrom order.Status
	reason     string
	buyer      int
	lines      []line
}

// orders are 30: 15 delivered, 4 cancelled, 3 handed to the courier, 2 packed, 2 accepted and 4 new.
// A delivered order takes about a day to come through, so even the newest delivered one is from yesterday.
var orders = []plannedOrder{
	{daysAgo: 28, hour: 10, final: order.StatusDelivered, buyer: 0, lines: []line{{"MN-JK1", 1}, {"HP-RB5", 1}}},
	{daysAgo: 26, hour: 14, final: order.StatusDelivered, buyer: 1, lines: []line{{"PH-A55", 1}}},
	{daysAgo: 24, hour: 11, final: order.StatusDelivered, buyer: 2, lines: []line{{"PH-RN13", 2}}},
	{daysAgo: 22, hour: 16, final: order.StatusDelivered, buyer: 3, lines: []line{{"MN-TS1", 3}, {"KD-SN1", 1}}},
	{daysAgo: 20, hour: 9, final: order.StatusDelivered, buyer: 4, lines: []line{{"LT-IP3", 1}}},
	{daysAgo: 17, hour: 13, final: order.StatusDelivered, buyer: 5, lines: []line{{"KT-PAN28", 2}, {"KT-KTL17", 1}}},
	{daysAgo: 14, hour: 15, final: order.StatusDelivered, buyer: 6, lines: []line{{"PH-IP13", 1}}},
	{daysAgo: 12, hour: 12, final: order.StatusDelivered, buyer: 7, lines: []line{{"MN-TS1", 2}, {"WM-DR1", 1}}},
	{daysAgo: 9, hour: 10, final: order.StatusDelivered, buyer: 0, lines: []line{{"PH-RN13", 1}, {"HP-RB5", 2}}},
	{daysAgo: 7, hour: 17, final: order.StatusDelivered, buyer: 1, lines: []line{{"FR-CH1", 1}}},
	{daysAgo: 5, hour: 11, final: order.StatusDelivered, buyer: 2, lines: []line{{"HP-AP2", 2}}},
	{daysAgo: 4, hour: 14, final: order.StatusDelivered, buyer: 3, lines: []line{{"MN-TS1", 4}, {"HP-RB5", 1}}},
	{daysAgo: 3, hour: 10, final: order.StatusDelivered, buyer: 4, lines: []line{{"PH-RN13", 2}}},
	{daysAgo: 2, hour: 16, final: order.StatusDelivered, buyer: 5, lines: []line{{"HP-RB5", 3}, {"KT-KTL17", 1}}},
	{daysAgo: 2, hour: 9, final: order.StatusDelivered, buyer: 6, lines: []line{{"PH-A55", 1}, {"HP-AP2", 1}}},

	{daysAgo: 19, hour: 12, final: order.StatusCancelled, cancelFrom: order.StatusNew, reason: "Buyer changed their mind", buyer: 7, lines: []line{{"PH-IP13", 1}}},
	{daysAgo: 13, hour: 15, final: order.StatusCancelled, cancelFrom: order.StatusAccepted, reason: "Delivery address is outside the delivery area", buyer: 3, lines: []line{{"KD-SN1", 2}}},
	{daysAgo: 6, hour: 10, final: order.StatusCancelled, cancelFrom: order.StatusPacked, reason: "Buyer did not answer the phone", buyer: 0, lines: []line{{"WM-DR1", 1}}},
	{daysAgo: 1, hour: 13, final: order.StatusCancelled, cancelFrom: order.StatusNew, reason: "Wrong address, the buyer asked to cancel", buyer: 6, lines: []line{{"KT-KTL17", 1}}},

	{daysAgo: 1, hour: 10, final: order.StatusHandedToCourier, buyer: 2, lines: []line{{"LT-IP3", 1}}},
	{daysAgo: 1, hour: 15, final: order.StatusHandedToCourier, buyer: 5, lines: []line{{"MN-JK1", 1}, {"MN-TS1", 2}}},
	{daysAgo: 2, hour: 12, final: order.StatusHandedToCourier, buyer: 4, lines: []line{{"PH-RN13", 1}}},

	{daysAgo: 1, hour: 11, final: order.StatusPacked, buyer: 1, lines: []line{{"HP-RB5", 2}}},
	{today: 0.25, final: order.StatusPacked, buyer: 7, lines: []line{{"PH-A55", 1}}},

	{today: 0.45, final: order.StatusAccepted, buyer: 3, lines: []line{{"MN-TS1", 2}, {"KT-PAN28", 1}}},
	{daysAgo: 1, hour: 16, final: order.StatusAccepted, buyer: 0, lines: []line{{"HP-AP2", 1}}},

	{today: 0.6, final: order.StatusNew, buyer: 6, lines: []line{{"PH-RN13", 1}}},
	{today: 0.75, final: order.StatusNew, buyer: 2, lines: []line{{"HP-RB5", 1}, {"KT-KTL17", 1}}},
	{today: 0.9, final: order.StatusNew, buyer: 1, lines: []line{{"FR-CH1", 1}}},
	{daysAgo: 1, hour: 18, final: order.StatusNew, buyer: 5, lines: []line{{"WM-DR1", 2}, {"MN-TS1", 1}}},
}
