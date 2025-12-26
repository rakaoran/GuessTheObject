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
	return "meow"
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
	return username + "ß" + hasher.Hash(username+mtm.key)
}

func (mtm *MockTokenManager) Verify(token string) (string, error) {
	pts := strings.Split(token, "ß")
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
	setup         func(repo *MockPlayerRepo)
}

type LoginTestCase struct {
	description   string
	username      string
	password      string
	expectedError error
}

func TestAuthService(t *testing.T) {

	var signupTests = []SignupTestCase{
		{
			description:   "normal",
			username:      "oussama145",
			password:      "12345678",
			expectedError: nil,
		},
		{
			description:   "with underscore",
			username:      "oussama145_two",
			password:      "12345678ermtrmt",
			expectedError: nil,
		},
		{
			description:   "dupplicate username",
			username:      "oussama145",
			password:      "12345678",
			expectedError: auth.UsernameAlreadyExistsErr,
			setup:         func(repo *MockPlayerRepo) { repo.CreatePlayer("oussama145", "16449976413") },
		},
		{
			description:   "dupplicate username with weak password",
			username:      "oussama145",
			password:      "12345",
			expectedError: auth.WeakPasswordErr,
			setup:         func(repo *MockPlayerRepo) { repo.CreatePlayer("oussama145", "16449976413") },
		},
		{
			description:   "short password",
			username:      "oussama",
			password:      "1234567",
			expectedError: auth.WeakPasswordErr,
		},
		{
			description:   "username too short",
			username:      "ou",
			password:      "12345678",
			expectedError: auth.InvalidUsernameFormatErr,
		},
		{
			description:   "username too long",
			username:      "oussamaermtermtermtermtrtmermterm",
			password:      "12345678",
			expectedError: auth.InvalidUsernameFormatErr,
		},
		{
			description:   "username with space",
			username:      "oussama_is the best",
			password:      "12345678",
			expectedError: auth.InvalidUsernameFormatErr,
		},
		{
			description:   "username withweird symbols",
			username:      "oussama-remt!#$@#$%^^&&*(()_++++====ß´í¯ß)",
			password:      "12345678",
			expectedError: auth.InvalidUsernameFormatErr,
		},
		{
			description:   "absent username",
			username:      "",
			password:      "12345678",
			expectedError: auth.InvalidUsernameFormatErr,
		},
		{
			description:   "absent password",
			username:      "oussama",
			password:      "",
			expectedError: auth.WeakPasswordErr,
		},
		{
			description:   "absent username and password",
			username:      "",
			password:      "",
			expectedError: auth.InvalidUsernameFormatErr,
		},
	}

	for _, tc := range signupTests {
		t.Run(tc.description, func(t *testing.T) {

			playerRepo := MockPlayerRepo{}
			passwordHasher := MockPasswordHasher{}
			tokenManager := MockTokenManager{key: "potato"}

			authService := auth.NewService(&playerRepo, &passwordHasher, &tokenManager)
			if tc.setup != nil {
				tc.setup(&playerRepo)
			}

			token, err := authService.Signup(tc.username, tc.password)

			assert.ErrorIs(t, tc.expectedError, err, tc.username, tc.password)

			if err == nil {
				assert.Equal(t, token, tokenManager.Generate(tc.username), "Asserting the service returns the correct token")
			}
		})

	}
}
