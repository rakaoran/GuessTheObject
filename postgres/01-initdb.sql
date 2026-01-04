CREATE TABLE users(
    id UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(15) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE isready();