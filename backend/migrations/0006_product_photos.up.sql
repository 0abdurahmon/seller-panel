CREATE TABLE product_photos (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint  NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        text    NOT NULL,
    position   integer NOT NULL,
    CONSTRAINT product_photos_url_check CHECK (btrim(url) <> ''),
    CONSTRAINT product_photos_position_check CHECK (position >= 0 AND position < 5),
    CONSTRAINT product_photos_position_key UNIQUE (product_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT product_photos_url_key UNIQUE (product_id, url)
);
