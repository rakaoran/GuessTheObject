package auth_test

import (
	"api/auth"
	"api/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockUserRepo struct {
	users               []*domain.User
	CreateUserFn        func(ctx context.Context, username string, passwordHash string) (string, error)
	GetUserByUsernameFn func(ctx context.Context, username string) (domain.User, error)
	GetUserByIdFn       func(ctx context.Context, id string) (domain.User, error)
}

func (mpr *MockUserRepo) CreateUser(ctx context.Context, username string, passwordHash string) (string, error) {
	uuid, err := mpr.CreateUserFn(ctx, username, passwordHash)
	if err != nil {
		return uuid, err
	}
	mpr.users = append(mpr.users, &domain.User{Id: uuid, Username: username, PasswordHash: passwordHash})
	return uuid, nil
}

func (mpr *MockUserRepo) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	return mpr.GetUserByUsernameFn(ctx, username)
}

func (mpr *MockUserRepo) GetUserById(ctx context.Context, id string) (domain.User, error) {
	return mpr.GetUserByIdFn(ctx, id)
}

type MockPasswordHasher struct {
	HashFn    func(password string) (string, error)
	CompareFn func(hash, password string) (bool, error)
}

func (mph *MockPasswordHasher) Hash(password string) (string, error) {
	return mph.HashFn(password)
}

func (mph *MockPasswordHasher) Compare(hash, password string) (bool, error) {
	return mph.CompareFn(hash, password)
}

type MockTokenManager struct {
	GenerateFn func(id string, now time.Time) (string, error)
	VerifyFn   func(token string) (string, error)
}

func (mtm *MockTokenManager) Generate(id string, now time.Time) (string, error) {
	return mtm.GenerateFn(id, now)
}

func (mtm *MockTokenManager) Verify(token string) (string, error) {
	return mtm.VerifyFn(token)
}

type SignupTestCase struct {
	description         string
	username            string
	password            string
	expectedError       error
	expectedCreatedUser *domain.User
	expectedToken       string
	CreateUserFn        func(ctx context.Context, username string, passwordHash string) (string, error)
	HashFn              func(password string) (string, error)
	GenerateFn          func(username string, now time.Time) (string, error)
}

func TestSignup(t *testing.T) {

	var signupTests = []SignupTestCase{
		{
			description:         "normal case",
			username:            "oussama",
			password:            "12345678",
			expectedError:       nil,
			expectedCreatedUser: &domain.User{Id: "111-111", Username: "oussama", PasswordHash: "1234567812345678"},
			expectedToken:       "111-111.tokkken",
			CreateUserFn:        func(ctx context.Context, username, passwordHash string) (string, error) { return "111-111", nil },
			HashFn:              func(password string) (string, error) { return password + password, nil },
			GenerateFn:          func(id string, now time.Time) (string, error) { return id + "." + "tokkken", nil },
		},
		{
			description:         "normal case, but hashing func exploded",
			username:            "oussama",
			password:            "12345678",
			expectedError:       domain.HashingError,
			expectedCreatedUser: nil,
			HashFn:              func(password string) (string, error) { return "", errors.New("argon2id bomb") },
		},
		{
			description:         "normal case, user created, but token generator func exploded",
			username:            "oussama",
			password:            "12345678",
			expectedError:       domain.TokenError,
			expectedCreatedUser: &domain.User{Id: "111-111", Username: "oussama", PasswordHash: "1234567812345678"},
			CreateUserFn:        func(ctx context.Context, username, passwordHash string) (string, error) { return "111-111", nil },
			HashFn:              func(password string) (string, error) { return password + password, nil },
			GenerateFn:          func(id string, now time.Time) (string, error) { return "", errors.New("internal entropy unavailable") },
		},
		{
			description:   "uppercase username",
			username:      "Oussama145",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:         "with underscore",
			username:            "oussama145_two",
			password:            "12345678ermtrmt",
			expectedError:       nil,
			expectedCreatedUser: &domain.User{Id: "111-111", Username: "oussama145_two", PasswordHash: "hash11"},
			expectedToken:       "111-111.tokkken",
			CreateUserFn:        func(ctx context.Context, username, passwordHash string) (string, error) { return "111-111", nil },
			HashFn:              func(password string) (string, error) { return "hash11", nil },
			GenerateFn:          func(id string, now time.Time) (string, error) { return id + "." + "tokkken", nil },
		},
		{
			description:         "dupplicate username",
			username:            "oussama145",
			password:            "12345678",
			expectedError:       domain.ErrDuplicateUsername,
			expectedCreatedUser: nil,
			CreateUserFn: func(ctx context.Context, username, passwordHash string) (string, error) {
				return "", domain.ErrDuplicateUsername
			},
			HashFn: func(password string) (string, error) { return "hash11", nil },
		},
		{
			description:   "short password",
			username:      "oussama",
			password:      "1234567",
			expectedError: auth.ErrWeakPassword,
		},
		{
			description:   "too long password",
			username:      "oussama",
			password:      "12345676444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444",
			expectedError: auth.ErrPasswordTooLong,
		},
		{
			description:   "username too short",
			username:      "ou",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username too long",
			username:      "oussamaermtermtermtermtrtmermterm",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username with space",
			username:      "oussama the best",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username withweird symbols",
			username:      "oussama-remt!#$@#$%^^&&*(()_++++====ß´í¯ß)",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "absent username",
			username:      "",
			password:      "12345678",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "absent password",
			username:      "oussama",
			password:      "",
			expectedError: auth.ErrWeakPassword,
		},
		{
			description:   "absent username and password",
			username:      "",
			password:      "",
			expectedError: auth.ErrInvalidUsernameFormat,
		},
	}

	for _, tc := range signupTests {
		passwordHasher := MockPasswordHasher{}
		tokenManager := MockTokenManager{}
		userRepo := MockUserRepo{}
		userRepo.CreateUserFn = tc.CreateUserFn
		tokenManager.GenerateFn = tc.GenerateFn
		passwordHasher.HashFn = tc.HashFn
		authService := auth.NewService(&userRepo, &passwordHasher, &tokenManager)

		t.Run(tc.description, func(t *testing.T) {
			ctx := context.Background()
			username := tc.username
			password := tc.password
			token, err := authService.Signup(ctx, username, password)

			assert.ErrorIs(t, err, tc.expectedError)
			assert.Equal(t, tc.expectedToken, token, "Token issued and token expected mismatch")

			if tc.expectedCreatedUser != nil {
				player := userRepo.users[len(userRepo.users)-1]
				assert.Equal(t, *tc.expectedCreatedUser, *player)

			}
		})

	}
}

type LoginTestCase struct {
	description         string
	username            string
	password            string
	expectedError       error
	expectedToken       string
	GetUserByUsernameFn func(ctx context.Context, username string) (domain.User, error)
	CompareFn           func(hash, password string) (bool, error)
	GenerateFn          func(username string, now time.Time) (string, error)
}

func TestLogin(t *testing.T) {
	var loginTests = []LoginTestCase{
		{
			description:   "successful login",
			username:      "oussama",
			password:      "12345678",
			expectedError: nil,
			expectedToken: "111.tokkken",
			GetUserByUsernameFn: func(ctx context.Context, username string) (domain.User, error) {
				if username != "oussama" {
					return domain.User{}, domain.ErrUsernameNotFound
				}
				return domain.User{Id: "111", Username: "oussama", PasswordHash: "1234567812345678"}, nil
			},
			CompareFn: func(hash, password string) (bool, error) {
				return password+password == hash, nil
			},
			GenerateFn: func(id string, now time.Time) (string, error) {
				return id + "." + "tokkken", nil
			},
		},
		{
			description:   "user not found",
			username:      "ghost",
			password:      "12345678",
			expectedError: domain.ErrUsernameNotFound,
			GetUserByUsernameFn: func(ctx context.Context, username string) (domain.User, error) {
				return domain.User{}, domain.ErrUsernameNotFound
			},
		},
		{
			description:   "incorrect password",
			username:      "oussama",
			password:      "wrong_pass",
			expectedError: auth.ErrIncorrectPassword,
			GetUserByUsernameFn: func(ctx context.Context, username string) (domain.User, error) {
				if username != "oussama" {
					return domain.User{}, domain.ErrUsernameNotFound
				}
				return domain.User{Id: "111", Username: "oussama", PasswordHash: "hashed_pw"}, nil
			},
			CompareFn: func(hash, password string) (bool, error) {
				return false, nil
			},
		},
		{
			description:   "password comparison fails (hashing error)",
			username:      "oussama",
			password:      "12345678",
			expectedError: domain.HashingError,
			GetUserByUsernameFn: func(ctx context.Context, username string) (domain.User, error) {
				if username != "oussama" {
					return domain.User{}, domain.ErrUsernameNotFound
				}
				return domain.User{Id: "111", Username: "oussama", PasswordHash: "hashed_pw"}, nil
			},
			CompareFn: func(hash, password string) (bool, error) {
				return false, errors.New("argon2id mem allocation failure")
			},
		},
		{
			description:   "token generation fails",
			username:      "oussama_yaqdane",
			password:      "12345678",
			expectedError: domain.TokenError,
			GetUserByUsernameFn: func(ctx context.Context, username string) (domain.User, error) {
				if username != "oussama_yaqdane" {
					return domain.User{}, domain.ErrUsernameNotFound
				}
				return domain.User{Id: "111", Username: "oussama_yaqdane", PasswordHash: "1234567812345678"}, nil
			},
			CompareFn: func(hash, password string) (bool, error) {
				return password+password == hash, nil
			},
			GenerateFn: func(id string, now time.Time) (string, error) {
				return "", errors.New("jwt signing error")
			},
		},
	}

	for _, tc := range loginTests {
		t.Run(tc.description, func(t *testing.T) {
			passwordHasher := MockPasswordHasher{}
			tokenManager := MockTokenManager{}
			userRepo := MockUserRepo{}

			userRepo.GetUserByUsernameFn = tc.GetUserByUsernameFn
			passwordHasher.CompareFn = tc.CompareFn
			tokenManager.GenerateFn = tc.GenerateFn

			authService := auth.NewService(&userRepo, &passwordHasher, &tokenManager)
			ctx := context.Background()

			token, err := authService.Login(ctx, tc.username, tc.password)

			assert.ErrorIs(t, err, tc.expectedError)

			assert.Equal(t, tc.expectedToken, token)

		})
	}
}
