package auth_test

import (
	"api/auth"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockPlayerRepo struct {
	players []*auth.Player
}

func (mpr *MockPlayerRepo) CreatePlayer(username string, passwordHash string) error {
	for _, p := range mpr.players {
		if p.Username == username {
			return auth.DuplicateUsernameRepoError
		}
	}
	mpr.players = append(mpr.players, &auth.Player{username, passwordHash})
	return nil
}

func (mpr *MockPlayerRepo) GetPlayerByUsername(username string) (auth.Player, error) {
	for _, p := range mpr.players {
		if p.Username == username {
			return *p, nil
		}
	}
	return auth.Player{}, auth.PlayerNotFoundRepoError
}

type MockPasswordHasher struct{}

func (mph *MockPasswordHasher) Hash(password string) string {
	arr := []rune(password)

	for i := range arr {
		arr[i] = arr[i] ^ 7 + 5
	}

	return string(arr)
}

func (mph *MockPasswordHasher) Compare(hash, password string) bool {
	hashedPassword := mph.Hash(password)
	return hashedPassword == hash
}

type MockTokenManager struct {
	key string
}

func (mtm *MockTokenManager) Generate(username string) string {
	hasher := MockPasswordHasher{}
	return username + "." + hasher.Hash(username+mtm.key)
}

func (mtm *MockTokenManager) Verify(token string) (string, error) {
	pts := strings.Split(token, ".")
	if len(pts) != 2 {
		return "", auth.InvalidTokenError
	}
	hasher := MockPasswordHasher{}
	if hasher.Hash(pts[0]+mtm.key) != pts[2] {
		return "", auth.InvalidTokenError
	}

	return pts[1], nil
}

// Separated since maybe we can add 'display name' in SignUp only...

type SignupTestCase struct {
	description   string
	username      string
	password      string
	expectedError error
}

type LoginTestCase struct {
	description   string
	username      string
	password      string
	expectedError error
}

func TestAuthService(t *testing.T) {
	playerRepo := MockPlayerRepo{}
	passwordHasher := MockPasswordHasher{}
	tokenManager := MockTokenManager{}

	authService := auth.NewService(&playerRepo, &passwordHasher, &tokenManager)

	var signupTests []SignupTestCase = []SignupTestCase{
		{"normal", "oussama145", "12345678", nil},
		{"with underscore", "oussama145_two", "12345678ermtrmt", nil},
		{"dupplicate username", "oussama145", "12345678", auth.UsernameAlreadyExistsErr},
		{"short password", "oussama", "1234567", auth.WeakPasswordErr},
		{"username too short", "ou", "12345678", auth.InvalidUsernameFormatErr},
		{"username too long", "oussamaermtermtermtermtrtmermterm", "12345678", auth.InvalidUsernameFormatErr},
		{"username with space", "oussama_is the best", "12345678", auth.InvalidUsernameFormatErr},
		{"with weird symbols", "oussama-remt!#$@#$%^^&&*(()_++++====ß´í¯ß)", "12345678", auth.InvalidUsernameFormatErr},
		{"absent username", "", "12345678", auth.InvalidUsernameFormatErr},
		{"absent password", "oussama", "", auth.WeakPasswordErr},
		{"absent username and password", "", "", auth.InvalidUsernameFormatErr},
	}

	for _, tc := range signupTests {
		_, err := authService.Signup(tc.username, tc.password)

		assert.ErrorIs(t, tc.expectedError, err, tc.description, tc.username, tc.password)

	}
}
