-- +goose Up
-- +goose StatementBegin
CREATE TABLE words(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word VARCHAR(50) UNIQUE NOT NULL
);

INSERT INTO words (word) VALUES
    ('elephant'),
    ('robot'),
    ('bicycle');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE words;
-- +goose StatementEnd
