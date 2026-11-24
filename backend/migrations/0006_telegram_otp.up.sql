ALTER TABLE sellers ADD COLUMN telegram_chat_id bigint UNIQUE;

-- One pending link per seller; only the hash of the token is stored.
CREATE TABLE telegram_link_tokens (
    token_hash text        PRIMARY KEY,
    seller_id  bigint      NOT NULL UNIQUE REFERENCES sellers (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

-- One active sign-in code per seller; only the hash of the code is stored.
CREATE TABLE login_codes (
    seller_id  bigint      PRIMARY KEY REFERENCES sellers (id) ON DELETE CASCADE,
    code_hash  text        NOT NULL,
    attempts   integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
