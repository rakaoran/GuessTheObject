package auth

import (
	"api/domain"
	"context"
	"time"
)

type AuthService interface {
	Signup(ctx context.Context, username, password string) (string, error)
	Login(ctx context.Context, username, password string) (string, error)
	VerifyToken(token string) (string, error)
	GenerateToken(id string) (string, error)
}

type UserRepo interface {
	CreateUser(ctx context.Context, username string, passwordHash string) (string, error)
	GetUserByUsername(ctx context.Context, username string) (domain.User, error)
	GetUserById(ctx context.Context, id string) (domain.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) (bool, error)
}

type TokenManager interface {
	Generate(id string, now time.Time) (string, error)
	Verify(token string) (string, error)
}
