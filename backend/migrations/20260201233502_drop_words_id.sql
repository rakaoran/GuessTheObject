-- +goose Up
-- +goose StatementBegin
BEGIN;
ALTER TABLE words DROP COLUMN id;
ALTER TABLE words ADD PRIMARY KEY (word);
COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
