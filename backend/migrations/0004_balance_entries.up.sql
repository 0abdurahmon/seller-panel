CREATE TYPE balance_entry_type AS ENUM ('sale', 'commission', 'payout');

-- order_id gets its foreign key to orders in the orders migration.
CREATE TABLE balance_entries (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shop_id    bigint             NOT NULL REFERENCES shops (id),
    order_id   bigint,
    type       balance_entry_type NOT NULL,
    amount     bigint             NOT NULL,
    created_at timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT balance_entries_sign CHECK (
        (type = 'sale' AND amount > 0)
        OR (type = 'commission' AND amount <= 0)
        OR (type = 'payout' AND amount < 0)
    ),
    CONSTRAINT balance_entries_order_matches_type CHECK ((type = 'payout') = (order_id IS NULL)),
    CONSTRAINT balance_entries_order_type_key UNIQUE (order_id, type)
);

CREATE INDEX balance_entries_shop_created_idx ON balance_entries (shop_id, created_at);
