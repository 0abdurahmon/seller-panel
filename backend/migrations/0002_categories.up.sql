CREATE TABLE categories (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name      text   NOT NULL CHECK (btrim(name) <> ''),
    parent_id bigint REFERENCES categories (id)
);

CREATE UNIQUE INDEX categories_parent_name_key ON categories (COALESCE(parent_id, 0), name);
CREATE INDEX categories_parent_idx ON categories (parent_id);

WITH top AS (
    INSERT INTO categories (name)
    VALUES ('Electronics'), ('Clothing'), ('Home')
    RETURNING id, name
)
INSERT INTO categories (name, parent_id)
SELECT child.name, top.id
FROM (VALUES
    ('Phones', 'Electronics'),
    ('Laptops', 'Electronics'),
    ('Headphones', 'Electronics'),
    ('Men', 'Clothing'),
    ('Women', 'Clothing'),
    ('Kids', 'Clothing'),
    ('Kitchen', 'Home'),
    ('Furniture', 'Home'),
    ('Decor', 'Home')
) AS child (name, parent_name)
JOIN top ON top.name = child.parent_name;
