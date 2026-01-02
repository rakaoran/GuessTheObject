package database

import (
	"api/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPlayerRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(ctx context.Context, connString string) (*PostgresPlayerRepo, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &PostgresPlayerRepo{pool: pool}, nil
}

func (pgur *PostgresPlayerRepo) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	player := domain.User{Username: username}

	row := pgur.pool.QueryRow(ctx, "SELECT password_hash FROM players WHERE username = $1", username)

	err := row.Scan(&player.PasswordHash)

	if err != nil {
		return player, domain.ErrUsernameNotFound
	}

	return player, nil
}

func (pgur *PostgresPlayerRepo) GetUserById(ctx context.Context, id string) (domain.User, error) {
	player := domain.User{Username: id}

	row := pgur.pool.QueryRow(ctx, "SELECT password_hash FROM players WHERE id = $1", id)

	err := row.Scan(&player.PasswordHash)

	if err != nil {
		return player, domain.ErrUsernameNotFound
	}

	return player, nil
}

func (pgur *PostgresPlayerRepo) CreateUser(ctx context.Context, username string, passwordHash string) (string, error) {
	row := pgur.pool.QueryRow(ctx, "INSERT INTO players(username, password_hash) VALUES($1, $2) RETURNING id", username, passwordHash)
	var id string
	err := row.Scan(&id)
	if err != nil {
		return "", domain.ErrDuplicateUsername
	}
	return id, nil
}
