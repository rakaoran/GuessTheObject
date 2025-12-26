CREATE TABLE players(
    username VARCHAR(15) PRIMARY KEY,
    password_hash TEXT,
    CONSTRAINT username_regex CHECK (username ~ '^[a-z0-9_]{3,20}$')
);

CREATE TABLE isready();