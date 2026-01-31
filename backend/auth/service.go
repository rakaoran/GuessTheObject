package auth

import (
	"context"
	"regexp"
	"time"
)

type authService struct {
	UserRepo       UserRepo
	passwordHasher PasswordHasher
	tokenManager   TokenManager
}

func NewService(userRepo UserRepo, passwordHasher PasswordHasher, tokenManager TokenManager) *authService {
	return &authService{userRepo, passwordHasher, tokenManager}
}

var usernameRegex = regexp.MustCompile("^[a-z0-9_]{3,20}$")

func validateUsernameFormat(username string) bool {
	match := usernameRegex.MatchString(username)
	return match
}

func (as *authService) Signup(ctx context.Context, username, password string) (string, error) {
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
		return "", err
	}

	id, err := as.UserRepo.CreateUser(ctx, username, passwordHash)
	if err != nil {
		return "", err
	}

	token, err := as.tokenManager.Generate(id, time.Now())
	if err != nil {
		return "", err
	}

	return token, nil
}

func (as *authService) Login(ctx context.Context, username, password string) (string, error) {
	player, err := as.UserRepo.GetUserByUsername(ctx, username)

	if err != nil {
		return "", err
	}

	match, err := as.passwordHasher.Compare(player.PasswordHash, password)

	if err != nil {
		return "", err
	}

	if !match {
		return "", ErrIncorrectPassword
	}
	token, err := as.tokenManager.Generate(player.Id, time.Now())
	if err != nil {
		return "", err
	}
	return token, nil
}

// VerifyToken returns the id if the token is valid, else, it returns an error
func (as *authService) VerifyToken(token string) (string, error) {
	return as.tokenManager.Verify(token)
}

func (as *authService) GenerateToken(id string) (string, error) {
	return as.tokenManager.Generate(id, time.Now())
}
