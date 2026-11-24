CREATE TYPE order_status AS ENUM ('new', 'accepted', 'packed', 'handed_to_courier', 'delivered', 'cancelled');

CREATE SEQUENCE order_number_seq START WITH 100000;

CREATE TABLE orders (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shop_id           bigint       NOT NULL REFERENCES shops (id),
    number            bigint       NOT NULL UNIQUE DEFAULT nextval('order_number_seq'),
    status            order_status NOT NULL DEFAULT 'new',
    buyer_name        text         NOT NULL,
    buyer_phone       text         NOT NULL,
    delivery_address  text         NOT NULL,
    total             bigint       NOT NULL,
    cancel_reason     text,
    created_at        timestamptz  NOT NULL DEFAULT now(),
    status_changed_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT orders_buyer_name_check CHECK (btrim(buyer_name) <> ''),
    CONSTRAINT orders_buyer_phone_check CHECK (btrim(buyer_phone) <> ''),
    CONSTRAINT orders_delivery_address_check CHECK (btrim(delivery_address) <> ''),
    CONSTRAINT orders_total_check CHECK (total > 0),
    CONSTRAINT orders_cancel_reason_matches_status CHECK (
        (status = 'cancelled' AND cancel_reason IS NOT NULL AND btrim(cancel_reason) <> '')
        OR (status <> 'cancelled' AND cancel_reason IS NULL)
    )
);

CREATE INDEX orders_shop_status_created_idx ON orders (shop_id, status, created_at);
CREATE INDEX orders_shop_created_idx ON orders (shop_id, created_at);
CREATE INDEX orders_shop_delivered_idx ON orders (shop_id, status_changed_at);

CREATE TABLE order_items (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id       bigint  NOT NULL REFERENCES orders (id),
    product_id     bigint  NOT NULL REFERENCES products (id),
    title_snapshot text    NOT NULL,
    price_snapshot bigint  NOT NULL,
    quantity       integer NOT NULL,
    CONSTRAINT order_items_price_check CHECK (price_snapshot > 0),
    CONSTRAINT order_items_quantity_check CHECK (quantity > 0),
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

ALTER TABLE balance_entries
    ADD CONSTRAINT balance_entries_order_id_fkey FOREIGN KEY (order_id) REFERENCES orders (id);
