CREATE TABLE users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(15) UNIQUE NOT NULL,
    password_hash TEXT,
    CONSTRAINT username_regex CHECK (username ~ '^[a-z0-9_]{3,20}$')
);

CREATE TABLE isready();