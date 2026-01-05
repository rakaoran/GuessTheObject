package database

import (
	"api/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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
	user := domain.User{Username: username}

	row := pgur.pool.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE username = $1", username)

	err := row.Scan(&user.Id, &user.PasswordHash)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return user, domain.ErrUserNotFound
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return domain.User{}, err
		default:
			return domain.User{}, fmt.Errorf("%w: %w", domain.UnexpectedDatabaseError, err)
		}
	}

	return user, nil
}

func (pgur *PostgresPlayerRepo) GetUserById(ctx context.Context, id string) (domain.User, error) {
	user := domain.User{Id: id}

	row := pgur.pool.QueryRow(ctx, "SELECT username, password_hash FROM users WHERE id = $1", id)

	err := row.Scan(&user.Username, &user.PasswordHash)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return domain.User{}, domain.ErrUserNotFound
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return domain.User{}, err

		default:
			return domain.User{}, fmt.Errorf("%w: %w", domain.UnexpectedDatabaseError, err)
		}
	}

	return user, nil
}

func (pgur *PostgresPlayerRepo) CreateUser(ctx context.Context, username string, passwordHash string) (string, error) {
	row := pgur.pool.QueryRow(ctx, "INSERT INTO users(username, password_hash) VALUES($1, $2) RETURNING id", username, passwordHash)

	var id string
	err := row.Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// "23505" is the PostgreSQL error code for unique_violation
			if pgErr.Code == "23505" {
				return "", domain.ErrDuplicateUsername
			}
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}

		return "", fmt.Errorf("%w: %w", domain.UnexpectedDatabaseError, err)
	}

	return id, nil
}
