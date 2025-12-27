CREATE TABLE players(
    username VARCHAR(15) PRIMARY KEY UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE isready();