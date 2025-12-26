package auth

import (
	"regexp"
	"unicode/utf8"
)

type service struct {
	playerRepo     PlayerRepo
	passwordHasher PasswordHasher
	tokenManager   TokenManager
}

func NewService(playerRepo PlayerRepo, passwordHasher PasswordHasher, tokenManager TokenManager) *service {
	return &service{playerRepo, passwordHasher, tokenManager}
}

func validateUsernameFormat(username string) bool {
	match, _ := regexp.MatchString("^[a-z0-9_]{3,20}$", username)
	return match
}

func (as *service) Signup(username, password string) (string, error) {
	if !validateUsernameFormat(username) {
		return "", InvalidUsernameFormatErr
	}

	if utf8.RuneCountInString(password) < 8 {
		return "", WeakPasswordErr
	}
	passwordHash := as.passwordHasher.Hash(password)

	err := as.playerRepo.CreatePlayer(username, passwordHash)

	if err != nil {
		switch err {
		case DuplicateUsernameRepoError:
			return "", UsernameAlreadyExistsErr
		default:
			return "", UnknownErr
		}
	}

	return as.tokenManager.Generate(username), nil
}

func (as *service) Login(username, password string) (string, error) {
	player, err := as.playerRepo.GetPlayerByUsername(username)

	if err != nil {
		return "", UsernameNotFoundErr
	}

	if !as.passwordHasher.Compare(player.PasswordHash, password) {
		return "", IncorrectPasswordErr
	}

	return as.tokenManager.Generate(player.Username), nil
}

// VerifyToken returns the username if the token is valid, else, it returns an error
func (as *service) VerifyToken(token string) (string, error) {
	return as.tokenManager.Verify(token)
}
