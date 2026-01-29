package storage

import "github.com/jackc/pgx/v5/pgxpool"

// GetPool returns the underlying connection pool.
// This is used by tests to query the database directly.
func (pgur *PostgresRepo) GetPool() *pgxpool.Pool {
	return pgur.pool
}
