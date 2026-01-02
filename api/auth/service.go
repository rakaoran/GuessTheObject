package auth

import (
	"api/domain"
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Signup errors
var (
	ErrWeakPassword          = errors.New("weak-password")
	ErrPasswordTooLong       = errors.New("weak-password")
	ErrInvalidUsernameFormat = errors.New("invalid-username-format")
)

// Login errors
var (
	ErrIncorrectPassword = errors.New("incorrect-password")
)

type AuthService struct {
	UserRepo       UserRepo
	passwordHasher PasswordHasher
	tokenManager   TokenManager
}

func NewService(userRepo UserRepo, passwordHasher PasswordHasher, tokenManager TokenManager) *AuthService {
	return &AuthService{userRepo, passwordHasher, tokenManager}
}

func validateUsernameFormat(username string) bool {
	match, _ := regexp.MatchString("^[a-z0-9_]{3,20}$", username)
	return match
}

func (as *AuthService) Signup(ctx context.Context, username, password string) (string, error) {
	if !validateUsernameFormat(username) {
		return "", ErrInvalidUsernameFormat
	}

	if len(password) < 8 {
		return "", ErrWeakPassword
	}

	if len(password) > 100 {
		return "", ErrPasswordTooLong
	}

	passwordHash, err := as.passwordHasher.Hash(password)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.HashingError, err)
	}

	id, err := as.UserRepo.CreateUser(ctx, username, passwordHash)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.DatabaseError, err)
	}

	token, err := as.tokenManager.Generate(id, time.Now())
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.TokenError, err)
	}

	return token, nil
}

func (as *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	player, err := as.UserRepo.GetUserByUsername(ctx, username)

	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.DatabaseError, err)
	}

	match, err := as.passwordHasher.Compare(player.PasswordHash, password)

	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.HashingError, err)
	}

	if !match {
		return "", ErrIncorrectPassword
	}
	token, err := as.tokenManager.Generate(player.Id, time.Now())
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.TokenError, err)
	}
	return token, nil
}

// VerifyToken returns the id if the token is valid, else, it returns an error
func (as *AuthService) VerifyToken(token string) (string, error) {
	return as.tokenManager.Verify(token)
}
