CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');

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
