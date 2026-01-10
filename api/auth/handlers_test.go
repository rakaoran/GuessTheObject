package auth_test

import (
	"api/auth"
	"api/domain"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuthService using testify/mock
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Signup(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) GenerateToken(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func TestSignupHandler(t *testing.T) {
	t.Parallel()

	type setupFn func(m *MockAuthService)

	type testCase struct {
		description   string
		body          string
		setupMocks    setupFn
		expectedCode  int
		expectedBody  string
		expectedToken string
	}

	exErr := errors.New("example error")
	gin.SetMode(gin.TestMode)

	testCases := []testCase{
		{
			description: "normal success",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").Return("tokenhaha", nil)
			},
			expectedCode:  http.StatusCreated,
			expectedBody:  "",
			expectedToken: "tokenhaha",
		},
		{
			description: "username already exists",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").Return("", domain.ErrDuplicateUsername)
			},
			expectedCode:  http.StatusConflict,
			expectedBody:  auth.ErrUsernameAlreadyExistsStr,
			expectedToken: "",
		},
		{
			description: "weak password",
			body:        `{"username":"oussama", "password":"123"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "123").Return("", auth.ErrWeakPassword)
			},
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrWeakPasswordStr,
			expectedToken: "",
		},
		{
			description: "password too long",
			body:        `{"username":"oussama", "password":"longpass"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "longpass").Return("", auth.ErrPasswordTooLong)
			},
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrPasswordTooLongStr,
			expectedToken: "",
		},
		{
			description: "invalid username format",
			body:        `{"username":"bad format", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "bad format", "pass1234").Return("", auth.ErrInvalidUsernameFormat)
			},
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidUsernameFormatStr,
			expectedToken: "",
		},
		{
			description:   "non json request",
			body:          `{`,
			setupMocks:    func(m *MockAuthService) {},
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidRequestFormatStr,
			expectedToken: "",
		},
		{
			description: "database failure (hashing error flow)",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").
					Return("", errors.Join(domain.UnexpectedDatabaseError, exErr))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "token generation failure",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").
					Return("", errors.Join(domain.UnexpectedTokenGenerationError, exErr))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrAccountCreatedButNoToken,
			expectedToken: "",
		},
		{
			description: "timeout error",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").Return("", context.DeadlineExceeded)
			},
			expectedCode:  http.StatusGatewayTimeout,
			expectedBody:  auth.ErrServerTimeoutStr,
			expectedToken: "",
		},
		{
			description: "client closed request",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Signup", mock.Anything, "oussama", "pass1234").Return("", context.Canceled)
			},
			expectedCode:  499,
			expectedBody:  "",
			expectedToken: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockService := new(MockAuthService)
			if tc.setupMocks != nil {
				tc.setupMocks(mockService)
			}

			authHandler := auth.NewAuthHandler(mockService, 197*time.Second)
			server := gin.New()
			server.POST("/signup", authHandler.SignupHandler)

			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			// Cookie Assertion
			cookies := res.Result().Cookies()
			token := ""
			if len(cookies) > 0 {
				assert.Equal(t, "token", cookies[0].Name, "Token cookie must be 'token'")
				assert.Equal(t, "/", cookies[0].Path, "Cookie path must be '/'")
				// Allow small delta for max age calculation if needed, but int match usually works
				assert.Equal(t, 197, cookies[0].MaxAge, "Cookie max age mismatch")
				token = cookies[0].Value
			}

			assert.Equal(t, tc.expectedCode, res.Code, "HTTP status code mismatch")
			assert.Equal(t, tc.expectedBody, res.Body.String())
			assert.Equal(t, tc.expectedToken, token)

			mockService.AssertExpectations(t)
		})
	}
}

func TestLoginHandler(t *testing.T) {
	t.Parallel()

	type setupFn func(m *MockAuthService)

	type testCase struct {
		description   string
		body          string
		setupMocks    setupFn
		expectedCode  int
		expectedBody  string
		expectedToken string
	}

	exErr := errors.New("example error")
	gin.SetMode(gin.TestMode)

	testCases := []testCase{
		{
			description: "successful login",
			body:        `{"username":"oussama", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "oussama", "pass1234").Return("loginToken123", nil)
			},
			expectedCode:  http.StatusOK,
			expectedBody:  "",
			expectedToken: "loginToken123",
		},
		{
			description: "user not found",
			body:        `{"username":"ghost", "password":"pass1234"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "ghost", "pass1234").Return("", domain.ErrUserNotFound)
			},
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  auth.ErrInvalidCredentialsStr,
			expectedToken: "",
		},
		{
			description: "incorrect password",
			body:        `{"username":"oussama", "password":"wrong"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "oussama", "wrong").Return("", auth.ErrIncorrectPassword)
			},
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  auth.ErrInvalidCredentialsStr,
			expectedToken: "",
		},
		{
			description:   "non json request",
			body:          `{`,
			setupMocks:    func(m *MockAuthService) {},
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidRequestFormatStr,
			expectedToken: "",
		},
		{
			description: "timeout error",
			body:        `{"username":"oussama", "password":"pass"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "oussama", "pass").Return("", context.DeadlineExceeded)
			},
			expectedCode:  http.StatusGatewayTimeout,
			expectedBody:  auth.ErrServerTimeoutStr,
			expectedToken: "",
		},
		{
			description: "database failure",
			body:        `{"username":"oussama", "password":"pass"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "oussama", "pass").
					Return("", errors.Join(domain.UnexpectedDatabaseError, exErr))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "unknown error",
			body:        `{"username":"oussama", "password":"pass"}`,
			setupMocks: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "oussama", "pass").Return("", errors.New("random stuff"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
	}

	for _, tc := range testCases {

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockService := new(MockAuthService)
			if tc.setupMocks != nil {
				tc.setupMocks(mockService)
			}

			authHandler := auth.NewAuthHandler(mockService, 100*time.Second)
			server := gin.New()
			server.POST("/login", authHandler.LoginHandler)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			cookies := res.Result().Cookies()
			token := ""
			if len(cookies) > 0 {
				assert.Equal(t, "token", cookies[0].Name)
				assert.Equal(t, "/", cookies[0].Path)
				assert.Equal(t, 100, cookies[0].MaxAge)
				token = cookies[0].Value
			}

			assert.Equal(t, tc.expectedCode, res.Code)
			assert.Equal(t, tc.expectedBody, res.Body.String())
			assert.Equal(t, tc.expectedToken, token)

			mockService.AssertExpectations(t)
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	t.Parallel()
	mockService := new(MockAuthService)
	authHandler := auth.NewAuthHandler(mockService, 4*time.Second)
	server := gin.New()

	server.POST("/logout", authHandler.LogoutHandler)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	cookies := res.Result().Cookies()
	require.NotEmpty(t, cookies, "Must be a cookie here")
	assert.Equal(t, "token", cookies[0].Name)
	assert.Less(t, cookies[0].MaxAge, 0, "token age must be negative so the cookie gets deleted")
}

func TestRequireAuthMiddleware(t *testing.T) {
	t.Parallel()

	// Helper to setup server
	setupServer := func(m *MockAuthService) *gin.Engine {
		authHandler := auth.NewAuthHandler(m, 15*time.Second)
		server := gin.New()
		server.Use(authHandler.RequireAuthMiddleware(1 * time.Millisecond))
		server.GET("/play", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
			ctx.String(http.StatusOK, ctx.GetString("id"))
		})
		return server
	}

	t.Run("missing cookie", func(t *testing.T) {
		m := new(MockAuthService)
		server := setupServer(m)

		req := httptest.NewRequest(http.MethodGet, "/play", nil)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
		assert.Equal(t, auth.ErrMissingTokenStr, res.Body.String())
	})

	t.Run("valid token", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("VerifyToken", "valid-token").Return("user-id-123", nil)
		server := setupServer(m)

		req := httptest.NewRequest(http.MethodGet, "/play", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "valid-token"})
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "user-id-123", res.Body.String())
		m.AssertExpectations(t)
	})

	t.Run("expired token", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("VerifyToken", "expired-token").Return("", domain.ErrExpiredToken)
		server := setupServer(m)

		req := httptest.NewRequest(http.MethodGet, "/play", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "expired-token"})
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
		assert.Equal(t, auth.ErrExpiredTokenStr, res.Body.String())
		m.AssertExpectations(t)
	})
}

func TestRefreshSessionHandler(t *testing.T) {
	t.Parallel()

	type setupFn func(m *MockAuthService)

	type testCase struct {
		description   string
		cookieValue   string
		setupMocks    setupFn
		expectedCode  int
		expectedBody  string
		expectedToken string
	}

	gin.SetMode(gin.TestMode)
	exErr := errors.New("example error")

	testCases := []testCase{
		{
			description:   "Missing token cookie",
			cookieValue:   "", // Empty implies no cookie set in request for this test logic
			setupMocks:    func(m *MockAuthService) {},
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  "unauthenticated",
			expectedToken: "",
		},
		{
			description: "Invalid or expired token",
			cookieValue: "bad-token",
			setupMocks: func(m *MockAuthService) {
				m.On("VerifyToken", "bad-token").Return("", domain.ErrExpiredToken)
			},
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  "bad-token",
			expectedToken: "",
		},
		{
			description: "Token generation failure",
			cookieValue: "valid-token",
			setupMocks: func(m *MockAuthService) {
				m.On("VerifyToken", "valid-token").Return("user-123", nil)
				m.On("GenerateToken", "user-123").Return("", errors.Join(domain.UnexpectedTokenGenerationError, exErr))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  "",
			expectedToken: "",
		},
		{
			description: "Successful refresh",
			cookieValue: "valid-token",
			setupMocks: func(m *MockAuthService) {
				m.On("VerifyToken", "valid-token").Return("user-123", nil)
				m.On("GenerateToken", "user-123").Return("new-refreshed-token", nil)
			},
			expectedCode:  http.StatusOK,
			expectedBody:  "",
			expectedToken: "new-refreshed-token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockService := new(MockAuthService)
			if tc.setupMocks != nil {
				tc.setupMocks(mockService)
			}

			authHandler := auth.NewAuthHandler(mockService, 24*time.Hour)
			server := gin.New()
			server.POST("/refresh", authHandler.RefreshSessionHandler)

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			if tc.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "token", Value: tc.cookieValue})
			}
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedCode, res.Code)
			assert.Equal(t, tc.expectedBody, res.Body.String())

			if tc.expectedToken != "" {
				cookies := res.Result().Cookies()
				if assert.NotEmpty(t, cookies, "Expected a response cookie but got none") {
					assert.Equal(t, "token", cookies[0].Name)
					assert.Equal(t, tc.expectedToken, cookies[0].Value)
				}
			} else {
				assert.Empty(t, res.Result().Cookies())
			}

			mockService.AssertExpectations(t)
		})
	}
}
