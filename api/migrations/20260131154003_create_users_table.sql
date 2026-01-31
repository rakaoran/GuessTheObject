-- +goose Up
-- +goose StatementBegin
CREATE TABLE users(
    id UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(15) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1; -- LOL NEVER DELETE THAT TABLE
-- +goose StatementEnd
