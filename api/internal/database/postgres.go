package database

import (
	"api/auth"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPlayerRepo struct {
	ctx  context.Context
	pool *pgxpool.Pool
}

func NewPostgresRepo(connString string) (*PostgresPlayerRepo, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &PostgresPlayerRepo{ctx: ctx, pool: pool}, nil
}

func (pgur *PostgresPlayerRepo) GetPlayerByUsername(username string) (auth.Player, error) {
	player := auth.Player{Username: username}

	row := pgur.pool.QueryRow(pgur.ctx, "SELECT password_hash FROM players WHERE username = $1", username)

	err := row.Scan(&player.PasswordHash)

	if err != nil {
		return player, auth.PlayerNotFoundRepoError
	}

	return player, nil
}

func (pgur *PostgresPlayerRepo) CreatePlayer(username string, passwordHash string) error {
	_, err := pgur.pool.Exec(pgur.ctx, "INSERT INTO players(username, password_hash) VALUES($1, $2)", username, passwordHash)
	if err != nil {
		return auth.DuplicateUsernameRepoError
	}
	return nil
}
