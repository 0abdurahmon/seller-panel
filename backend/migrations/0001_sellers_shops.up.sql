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
