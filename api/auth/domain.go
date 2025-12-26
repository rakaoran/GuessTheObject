package auth

import "errors"

type PlayerRepo interface {
	CreatePlayer(username string, passwordHash string) error
	GetPlayerByUsername(username string) (Player, error)
}

var (
	DuplicateUsernameRepoError = errors.New("duplicate-username")
	PlayerNotFoundRepoError    = errors.New("player-not-found")
)

type PasswordHasher interface {
	Hash(password string) string
	Compare(hash, password string) bool
}

type TokenManager interface {
	Generate(username string) string
	Verify(token string) (string, error)
}

var (
	InvalidTokenTokenManagerError = errors.New("invalid-token")
)

type Player struct {
	Username     string
	PasswordHash string
}
