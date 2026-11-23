# Domain notes

This is a **suggested** model. You may change it — but write down why in `DECISIONS.md`.

## Entities

### Seller
`id, phone/email, password_hash (if used), created_at`

### Shop (1 per seller)
`id, seller_id, name, logo_url, description, contact_phone, pickup_address, created_at`

### Category (seeded, read‑only for sellers)
`id, name, parent_id?`
Seed a small tree, e.g. Electronics › Phones, Clothing › Men / Women, Home › Kitchen.

### Product
`id, shop_id, category_id, title, description, price (integer, in so'm), sku, stock, reserved, status (draft | active | archived), photos[], created_at, updated_at`

- `available = stock − reserved` — this is what buyers can order.
- `available == 0` → shown as **Out of stock**.
- `available ≤ low_stock_threshold` → shown as **Low stock**.
- Price is an **integer** (no floats for money).
- `sku` is unique per shop.

### Order
`id, shop_id, number (human‑readable, e.g. #100245), status, buyer_name, buyer_phone, delivery_address, items[], total, cancel_reason?, created_at, status_changed_at`

### OrderItem
`id, order_id, product_id, title_snapshot, price_snapshot, quantity`
Snapshot title and price at order time — editing a product later must not change old orders.

### BalanceEntry
`id, shop_id, order_id, type (sale | commission | payout), amount, created_at`

## Order status machine

```
          ┌──────────── cancelled ◄──────────┐
          │                ▲                 │
new ──► accepted ──► packed ──► handed_to_courier ──► delivered
```

| From | Allowed next | Side effect |
|------|--------------|-------------|
| `new` | `accepted`, `cancelled` | accept → `reserved += qty` (fail if not enough available) |
| `accepted` | `packed`, `cancelled` | cancel → `reserved −= qty` |
| `packed` | `handed_to_courier`, `cancelled` | cancel → `reserved −= qty` |
| `handed_to_courier` | `delivered` | — |
| `delivered` | — (final) | `stock −= qty`, `reserved −= qty`; add `sale` (+total) and `commission` (−10% of total) balance entries |
| `cancelled` | — (final) | reason is required |

Any other transition must be rejected by the **backend** (not only hidden in the UI).

## Fake buyer orders

Provide one of:
- a seed script that creates ~30 orders in different statuses over the last 30 days, **and**
- a "Simulate order" button / endpoint (dev only) that creates a random `new` order from the shop's active products.

## Suggested API (REST — GraphQL/tRPC also fine)

```
POST   /auth/request-code        POST /auth/verify          POST /auth/logout
GET    /me                       GET/PUT /shop
GET    /categories
GET    /products?q=&category=&status=&page=
POST   /products                 GET/PUT /products/:id
PATCH  /products/:id/stock       POST /products/:id/archive
POST   /uploads                  (photos)
GET    /orders?status=&page=     GET /orders/:id
POST   /orders/:id/transition    { "to": "accepted", "reason"?: "..." }
GET    /dashboard/summary
GET    /balance                  GET /balance/entries?page=
POST   /dev/simulate-order       (dev only)
```

Agree on the API contract on Day 1 (an OpenAPI file, a shared types package, or a markdown table) so frontend and backend can work in parallel against mocks.
