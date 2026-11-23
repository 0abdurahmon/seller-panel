# Database schema

PostgreSQL schema for the Seller Panel as SQL, plus the key queries that enforce the rules. Entities and rules are explained in [`DATA_MODEL.md`](DATA_MODEL.md), conventions in [`ARCHITECTURE.md`](ARCHITECTURE.md). Status: **draft for team review** (issue #31).

**What was verified:** the schema below was applied to PostgreSQL 17 (the version in `docker-compose.yml`), 18 invalid inserts and updates were each rejected by the intended constraint, and the reserve statement in section 3 was run, including 6 concurrent reserves of 2 units against a stock of 10: exactly 5 succeeded and `reserved` never passed 10. The other queries in section 3 are designs and get tested together with the feature that uses them.

## 1. Conventions

- IDs: `bigint GENERATED ALWAYS AS IDENTITY`. Money: `bigint` in so'm. Time: `timestamptz`.
- Status columns are PostgreSQL enums, so a typo cannot be stored.
- Rules that must never break are `CHECK` or `UNIQUE` constraints in the database, not only checks in Go. The service validates first and returns a friendly error, the database is the last line of defence.
- No rows are deleted in normal operation. Products are archived, orders are cancelled, balance entries are append only. The only cascade is `product_photos` when a product is removed.
- Every shop-owned table carries `shop_id` (directly or through its parent) so queries can always be scoped to one shop.

## 2. Schema

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');
CREATE TYPE order_status AS ENUM ('new', 'accepted', 'packed', 'handed_to_courier', 'delivered', 'cancelled');
CREATE TYPE balance_entry_type AS ENUM ('sale', 'commission', 'payout');

CREATE TABLE sellers (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL UNIQUE CHECK (email = lower(email) AND email <> ''),
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE shops (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seller_id      bigint      NOT NULL UNIQUE REFERENCES sellers (id),
    name           text        NOT NULL CHECK (btrim(name) <> ''),
    logo_url       text,
    description    text        NOT NULL DEFAULT '',
    contact_phone  text,
    pickup_address text,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name      text   NOT NULL CHECK (btrim(name) <> ''),
    parent_id bigint REFERENCES categories (id)
);
CREATE UNIQUE INDEX categories_parent_name_key ON categories (COALESCE(parent_id, 0), name);
CREATE INDEX categories_parent_idx ON categories (parent_id);

CREATE TABLE products (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shop_id     bigint         NOT NULL REFERENCES shops (id),
    category_id bigint         NOT NULL REFERENCES categories (id),
    title       text           NOT NULL CHECK (btrim(title) <> ''),
    description text           NOT NULL DEFAULT '',
    price       bigint         NOT NULL CHECK (price > 0),
    sku         text           NOT NULL CHECK (btrim(sku) <> ''),
    stock       integer        NOT NULL DEFAULT 0 CHECK (stock >= 0),
    reserved    integer        NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    status      product_status NOT NULL DEFAULT 'draft',
    created_at  timestamptz    NOT NULL DEFAULT now(),
    updated_at  timestamptz    NOT NULL DEFAULT now(),
    CONSTRAINT products_reserved_within_stock CHECK (reserved <= stock),
    CONSTRAINT products_shop_sku_key UNIQUE (shop_id, sku)
);
CREATE INDEX products_shop_status_created_idx ON products (shop_id, status, created_at DESC);
CREATE INDEX products_shop_category_idx ON products (shop_id, category_id);
CREATE INDEX products_title_trgm_idx ON products USING gin (title gin_trgm_ops);
CREATE INDEX products_sku_trgm_idx ON products USING gin (sku gin_trgm_ops);

CREATE TABLE product_photos (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint  NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        text    NOT NULL CHECK (url <> ''),
    position   integer NOT NULL CHECK (position >= 0),
    CONSTRAINT product_photos_position_key UNIQUE (product_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE SEQUENCE order_number_seq START WITH 100000;

CREATE TABLE orders (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shop_id           bigint       NOT NULL REFERENCES shops (id),
    number            bigint       NOT NULL UNIQUE DEFAULT nextval('order_number_seq'),
    status            order_status NOT NULL DEFAULT 'new',
    buyer_name        text         NOT NULL CHECK (btrim(buyer_name) <> ''),
    buyer_phone       text         NOT NULL CHECK (btrim(buyer_phone) <> ''),
    delivery_address  text         NOT NULL CHECK (btrim(delivery_address) <> ''),
    total             bigint       NOT NULL CHECK (total > 0),
    cancel_reason     text,
    created_at        timestamptz  NOT NULL DEFAULT now(),
    status_changed_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT orders_cancel_reason_matches_status CHECK (
        (status = 'cancelled' AND cancel_reason IS NOT NULL AND btrim(cancel_reason) <> '')
        OR (status <> 'cancelled' AND cancel_reason IS NULL)
    )
);
CREATE INDEX orders_shop_status_created_idx ON orders (shop_id, status, created_at DESC);
CREATE INDEX orders_shop_created_idx ON orders (shop_id, created_at DESC);
CREATE INDEX orders_shop_delivered_idx ON orders (shop_id, status_changed_at) WHERE status = 'delivered';

CREATE TABLE order_items (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id       bigint  NOT NULL REFERENCES orders (id),
    product_id     bigint  NOT NULL REFERENCES products (id),
    title_snapshot text    NOT NULL,
    price_snapshot bigint  NOT NULL CHECK (price_snapshot > 0),
    quantity       integer NOT NULL CHECK (quantity > 0),
    CONSTRAINT order_items_order_product_key UNIQUE (order_id, product_id)
);
CREATE INDEX order_items_product_idx ON order_items (product_id);

CREATE TABLE order_status_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    bigint       NOT NULL REFERENCES orders (id),
    from_status order_status,
    to_status   order_status NOT NULL,
    reason      text,
    created_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX order_status_history_order_idx ON order_status_history (order_id, created_at);

CREATE TABLE balance_entries (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shop_id    bigint             NOT NULL REFERENCES shops (id),
    order_id   bigint             REFERENCES orders (id),
    type       balance_entry_type NOT NULL,
    amount     bigint             NOT NULL,
    created_at timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT balance_entries_sign CHECK (
        (type = 'sale' AND amount > 0)
        OR (type = 'commission' AND amount <= 0)
        OR (type = 'payout' AND amount < 0)
    ),
    CONSTRAINT balance_entries_order_matches_type CHECK ((type = 'payout') = (order_id IS NULL))
);
CREATE UNIQUE INDEX balance_entries_order_type_key ON balance_entries (order_id, type) WHERE order_id IS NOT NULL;
CREATE INDEX balance_entries_shop_created_idx ON balance_entries (shop_id, created_at DESC);
```

### Which constraint protects which rule

| Rule | Enforced by |
|------|-------------|
| Stock never negative, reserved never negative or above stock | `stock >= 0`, `reserved >= 0`, `products_reserved_within_stock` |
| SKU unique per shop | `products_shop_sku_key` |
| One shop per seller, unique email (lower-case) | `shops.seller_id UNIQUE`, `sellers.email` check and unique |
| Price and quantity positive, money is an integer | `CHECK (price > 0)`, `CHECK (quantity > 0)`, `bigint` columns |
| Cancel reason required exactly when cancelled | `orders_cancel_reason_matches_status` |
| Only known statuses | enums |
| A product appears once per order | `order_items_order_product_key` |
| One `sale` and one `commission` entry per order, so a second delivery cannot double-count | `balance_entries_order_type_key` |
| Balance signs are right, payouts have no order | `balance_entries_sign`, `balance_entries_order_matches_type` |
| Order numbers are unique and human readable | `order_number_seq` from 100000, `number UNIQUE` |
| Photo positions do not collide | `product_photos_position_key` (deferrable, so reordering inside one transaction works) |

A negative `stock` update is reported by `products_reserved_within_stock` first, because `-1` is also below `reserved`. Either way the write is rejected.

## 3. Key queries

`$n` are bound parameters (pgx). Every query that touches shop data includes `shop_id`.

**Reserve stock** when an order is accepted (run per item inside the transaction, items ordered by `product_id`). Zero rows updated means `409 insufficient_stock` and the transaction rolls back:

```sql
UPDATE products
SET reserved = reserved + $qty, updated_at = now()
WHERE id = $product_id AND shop_id = $shop_id AND stock - reserved >= $qty;
```

**Release stock** when an accepted or packed order is cancelled:

```sql
UPDATE products
SET reserved = reserved - $qty, updated_at = now()
WHERE id = $product_id AND shop_id = $shop_id;
```

**Commit stock** when an order is delivered:

```sql
UPDATE products
SET stock = stock - $qty, reserved = reserved - $qty, updated_at = now()
WHERE id = $product_id AND shop_id = $shop_id;
```

**Change order status**: lock the order, let the service validate the transition, then update and log it in the same transaction:

```sql
SELECT id, status FROM orders WHERE id = $order_id AND shop_id = $shop_id FOR UPDATE;

UPDATE orders
SET status = $to, cancel_reason = $reason, status_changed_at = now()
WHERE id = $order_id;

INSERT INTO order_status_history (order_id, from_status, to_status, reason)
VALUES ($order_id, $from, $to, $reason);
```

**Balance entries on delivery** (`$commission = total * percent / 100`, integer division, computed in Go):

```sql
INSERT INTO balance_entries (shop_id, order_id, type, amount) VALUES
  ($shop_id, $order_id, 'sale', $total),
  ($shop_id, $order_id, 'commission', -$commission);
```

**Product list** with search, filters and pagination (`$q` may be null; trigram indexes serve `ILIKE`):

```sql
SELECT id, title, price, sku, stock, reserved, status, category_id
FROM products
WHERE shop_id = $shop_id
  AND ($status::product_status IS NULL OR status = $status)
  AND ($category_id::bigint IS NULL OR category_id = $category_id)
  AND ($q::text IS NULL OR title ILIKE '%' || $q || '%' OR sku ILIKE '%' || $q || '%')
ORDER BY created_at DESC, id DESC
LIMIT $page_size OFFSET $offset;
```

**Dashboard** (`$from` and `$to` are computed in the business time zone):

```sql
SELECT status, count(*) FROM orders
WHERE shop_id = $shop_id AND status IN ('new', 'accepted', 'packed')
GROUP BY status;

SELECT count(*) FROM orders
WHERE shop_id = $shop_id AND created_at >= $today_start AND created_at < $today_end;

SELECT COALESCE(sum(total), 0) FROM orders
WHERE shop_id = $shop_id AND status = 'delivered'
  AND status_changed_at >= $from AND status_changed_at < $to;

SELECT oi.product_id, oi.title_snapshot, sum(oi.quantity) AS units
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE o.shop_id = $shop_id AND o.status = 'delivered'
  AND o.status_changed_at >= $from AND o.status_changed_at < $to
GROUP BY oi.product_id, oi.title_snapshot
ORDER BY units DESC, oi.product_id
LIMIT 5;
```

**Balance** and history:

```sql
SELECT COALESCE(sum(amount), 0) FROM balance_entries WHERE shop_id = $shop_id;

SELECT id, order_id, type, amount, created_at FROM balance_entries
WHERE shop_id = $shop_id
ORDER BY created_at DESC, id DESC
LIMIT $page_size OFFSET $offset;
```

## 4. Indexes and the queries they serve

| Index | Serves |
|-------|--------|
| `products_shop_status_created_idx` | product list filtered by status, newest first |
| `products_shop_category_idx` | filter by category |
| `products_title_trgm_idx`, `products_sku_trgm_idx` | `ILIKE '%text%'` search |
| `orders_shop_status_created_idx` | order list by status, waiting-for-action counts |
| `orders_shop_created_idx` | all orders newest first, today's orders |
| `orders_shop_delivered_idx` (partial) | revenue and top products for delivered orders in a date range |
| `order_items_product_idx` | top products, product lookups |
| `balance_entries_shop_created_idx` | balance history |
| `order_status_history_order_idx` | order timeline |

## 5. Migrations and seed

- Files in `backend/migrations`, applied by golang-migrate on backend start (see `ARCHITECTURE.md`).
- `0001_init`: extension, enums, all tables and indexes above. Its `down` drops them in reverse order.
- `0002_categories`: the seeded category tree. Categories are reference data the app cannot work without, so they ship as a migration, not as demo data.
- Demo data (seller, shop, about 15 products, about 30 orders with balance entries) comes from `cmd/seed`, which only runs when asked. It writes through the same service code so the numbers stay consistent.
- A merged migration is never edited, a change is a new migration.

## 6. Known limits

- `order_items` does not check in SQL that a product belongs to the same shop as the order. The service enforces it because the simulate endpoint only picks the shop's own products. A composite foreign key would add `shop_id` to several tables for little gain.
- `OFFSET` pagination is fine at this size. Keyset pagination is the upgrade if lists grow large.

## 7. Open questions for the team

1. **Search.** `pg_trgm` gives fast partial matches on title and SKU. The alternative is plain `ILIKE` without the two GIN indexes, which is fine for a small demo. *Recommend:* keep trigram, the cost is one extension line.
2. **Categories as a migration or a seed command.** *Recommend:* migration (reference data).
3. **Same-shop check for order items** in SQL (composite key) or only in the service. *Recommend:* service only, as in section 6.
4. **`product_photos.position` deferrable unique key.** Slightly more complex but lets the UI reorder photos in one transaction. Keep it or use a plain unique key. *Recommend:* keep.
5. Answers to the open questions in `DATA_MODEL.md` (commission rounding, status history, low-stock threshold) change this schema only if the team picks a non-recommended option: dropping the history table or adding `shops.low_stock_threshold`.
