CREATE TABLE users(
    id UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(15) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE words(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word VARCHAR(50) UNIQUE NOT NULL
);

-- Insert initial words
INSERT INTO words (word) VALUES
    ('elephant'),
    ('rainbow'),
    ('guitar'),
    ('mountain'),
    ('butterfly'),
    ('telescope'),
    ('skateboard'),
    ('dinosaur'),
    ('lighthouse'),
    ('volcano'),
    ('penguin'),
    ('castle'),
    ('pizza'),
    ('rocket'),
    ('octopus'),
    ('treasure'),
    ('waterfall'),
    ('dragon'),
    ('robot'),
    ('bicycle');

CREATE TABLE isready();