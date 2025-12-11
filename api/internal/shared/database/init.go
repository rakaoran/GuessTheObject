package database

import (
	"api/internal/shared/configs"
	"api/internal/shared/logger"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool
var ctx = context.Background()

func Initialize() {
	_pool, err := pgxpool.New(ctx, configs.Envs.POSTGRES_URL)

	if err != nil {
		logger.Fatalf("Couldn't create pool: %v\n", err)
	}

	if err := _pool.Ping(ctx); err != nil {
		logger.Fatalf("Couldn't connect to database: %v\n", err)
	}
	pool = _pool
}
