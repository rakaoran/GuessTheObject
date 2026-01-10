package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"api/auth"
	"api/domain"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) CreateUser(ctx context.Context, username, passwordHash string) (string, error) {
	args := m.Called(ctx, username, passwordHash)
	return args.String(0), args.Error(1)
}

func (m *MockUserRepo) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserRepo) GetUserById(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.Error(1)
}

type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) Compare(hash, password string) (bool, error) {
	args := m.Called(hash, password)
	return args.Bool(0), args.Error(1)
}

type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) Generate(id string, now time.Time) (string, error) {
	args := m.Called(id, now)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) Verify(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func TestSignup(t *testing.T) {
	t.Parallel()

	type setupFn func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager)

	type testCase struct {
		description   string
		username      string
		password      string
		setupMocks    setupFn
		expectedToken string
		expectedError error
	}

	exampleErr := errors.New("example")

	testCases := []testCase{
		{
			description: "normal case",
			username:    "oussama",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				h.On("Hash", "12345678").Return("hashed_secret", nil)
				r.On("CreateUser", mock.Anything, "oussama", "hashed_secret").Return("111-111", nil)
				tm.On("Generate", "111-111", mock.AnythingOfType("time.Time")).Return("111-111.tokkken", nil)
			},
			expectedToken: "111-111.tokkken",
			expectedError: nil,
		},
		{
			description: "normal case, but hashing func exploded",
			username:    "oussama",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				h.On("Hash", "12345678").Return("", errors.Join(domain.UnexpectedPasswordHashingError, exampleErr))
			},
			expectedToken: "",
			expectedError: domain.UnexpectedPasswordHashingError,
		},
		{
			description: "normal case, user created, but token generator func exploded",
			username:    "oussama",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				h.On("Hash", "12345678").Return("hashed_secret", nil)
				r.On("CreateUser", mock.Anything, "oussama", "hashed_secret").Return("111-111", nil)

				// Token generation fails
				tm.On("Generate", "111-111", mock.Anything).Return("", errors.Join(domain.UnexpectedTokenGenerationError, exampleErr))
			},
			expectedToken: "",
			expectedError: domain.UnexpectedTokenGenerationError,
		},
		{
			description:   "uppercase username (validation fail)",
			username:      "Oussama145",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description: "duplicate username",
			username:    "oussama145",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				h.On("Hash", "12345678").Return("hashed_secret", nil)
				r.On("CreateUser", mock.Anything, "oussama145", "hashed_secret").Return("", domain.ErrDuplicateUsername)
			},
			expectedError: domain.ErrDuplicateUsername,
		},
		{
			description:   "short password",
			username:      "oussama",
			password:      "1234567",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrWeakPassword,
		},
		{
			description:   "too long password",
			username:      "oussama",
			password:      "12345676444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrPasswordTooLong,
		},
		{
			description:   "username too short",
			username:      "ou",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username with new lines",
			username:      "ohsktu\nyohoo\n",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username with new lines and weird stuff",
			username:      "oeermtu\nyohoo\nretemr3$#%",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username too long",
			username:      "oussamaermtermtermtermtrtmermterm",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "username with weird symbols",
			username:      "oussama-remt!#$@#$%^^&&*(()_++++====ß´í¯ß)",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "absent username",
			username:      "",
			password:      "12345678",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
		{
			description:   "absent password",
			username:      "oussama",
			password:      "",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrWeakPassword,
		},
		{
			description:   "absent username and password",
			username:      "",
			password:      "",
			setupMocks:    func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {},
			expectedError: auth.ErrInvalidUsernameFormat,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockUserRepo)
			mockHasher := new(MockPasswordHasher)
			mockToken := new(MockTokenManager)

			if tc.setupMocks != nil {
				tc.setupMocks(mockRepo, mockHasher, mockToken)
			}

			authService := auth.NewService(mockRepo, mockHasher, mockToken)
			token, err := authService.Signup(context.Background(), tc.username, tc.password)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedToken, token)

			// 5. Verify that expected calls (and ONLY expected calls) happened
			mockRepo.AssertExpectations(t)
			mockHasher.AssertExpectations(t)
			mockToken.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()

	type setupFn func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager)

	type testCase struct {
		description   string
		username      string
		password      string
		setupMocks    setupFn
		expectedToken string
		expectedError error
	}

	exampleErr := errors.New("example")

	testCases := []testCase{
		{
			description: "successful login",
			username:    "oussama",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				r.On("GetUserByUsername", mock.Anything, "oussama").
					Return(domain.User{Id: "111", Username: "oussama", PasswordHash: "hashed_secret"}, nil)
				h.On("Compare", "hashed_secret", "12345678").Return(true, nil)
				tm.On("Generate", "111", mock.AnythingOfType("time.Time")).
					Return("111.tokkken", nil)
			},
			expectedToken: "111.tokkken",
			expectedError: nil,
		},
		{
			description: "user not found",
			username:    "ghost",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				r.On("GetUserByUsername", mock.Anything, "ghost").
					Return(domain.User{}, domain.ErrUserNotFound)
			},
			expectedToken: "",
			expectedError: domain.ErrUserNotFound,
		},
		{
			description: "incorrect password",
			username:    "oussama",
			password:    "wrong_pass",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				r.On("GetUserByUsername", mock.Anything, "oussama").
					Return(domain.User{Id: "111", PasswordHash: "hashed_secret"}, nil)
				h.On("Compare", "hashed_secret", "wrong_pass").Return(false, nil)
			},
			expectedToken: "",
			expectedError: auth.ErrIncorrectPassword,
		},
		{
			description: "password comparison fails (hashing error)",
			username:    "oussama",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				r.On("GetUserByUsername", mock.Anything, "oussama").
					Return(domain.User{Id: "111", PasswordHash: "hashed_secret"}, nil)
				h.On("Compare", "hashed_secret", "12345678").
					Return(false, errors.Join(domain.UnexpectedPasswordHashingError, exampleErr))
			},
			expectedToken: "",
			expectedError: domain.UnexpectedPasswordHashingError,
		},
		{
			description: "token generation fails",
			username:    "oussama_yaqdane",
			password:    "12345678",
			setupMocks: func(r *MockUserRepo, h *MockPasswordHasher, tm *MockTokenManager) {
				r.On("GetUserByUsername", mock.Anything, "oussama_yaqdane").
					Return(domain.User{Id: "111", PasswordHash: "hashed_secret"}, nil)

				h.On("Compare", "hashed_secret", "12345678").Return(true, nil)
				tm.On("Generate", "111", mock.Anything).
					Return("", errors.Join(domain.UnexpectedTokenGenerationError, exampleErr))
			},
			expectedToken: "",
			expectedError: domain.UnexpectedTokenGenerationError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockRepo := new(MockUserRepo)
			mockHasher := new(MockPasswordHasher)
			mockToken := new(MockTokenManager)

			if tc.setupMocks != nil {
				tc.setupMocks(mockRepo, mockHasher, mockToken)
			}

			authService := auth.NewService(mockRepo, mockHasher, mockToken)

			token, err := authService.Login(context.Background(), tc.username, tc.password)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedToken, token)

			mockRepo.AssertExpectations(t)
			mockHasher.AssertExpectations(t)
			mockToken.AssertExpectations(t)
		})
	}
}
