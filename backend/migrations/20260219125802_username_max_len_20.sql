-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN username TYPE VARCHAR(20);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
