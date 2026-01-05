package auth_test

import (
	"api/auth"
	"api/domain"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAuthService struct {
	SignupFunc        func(ctx context.Context, username, password string) (string, error)
	LoginFunc         func(ctx context.Context, username, password string) (string, error)
	VerifyTokenFunc   func(token string) (string, error)
	GenerateTokenFunc func(id string) (string, error)
}

func (m *MockAuthService) Signup(ctx context.Context, username, password string) (string, error) {
	return m.SignupFunc(ctx, username, password)
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (string, error) {
	return m.LoginFunc(ctx, username, password)
}

func (m *MockAuthService) VerifyToken(token string) (string, error) {
	return m.VerifyTokenFunc(token)
}

func (m *MockAuthService) GenerateToken(id string) (string, error) {
	return m.GenerateTokenFunc(id)
}

// This function tests if the signup handler correctly passes the credentials to
// the signup service, and doesn't somehow create a user with username:pass1234 and password: oussama
// Other tests like http codes and errors returned shall be tested not here
func TestVariablePassingSignupHandler(t *testing.T) {
	t.Parallel()
	authService := MockAuthService{}
	var passedContext context.Context
	authService.SignupFunc = func(ctx context.Context, username, password string) (string, error) {
		passedContext = ctx
		return username + "." + password, nil
	}

	authHandler := auth.NewAuthHandler(&authService, 50)

	server := gin.New()
	server.POST("/signup", authHandler.SignupHandler)
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{"username":"oussama", "password":"pass1234"}`))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	username_password := strings.Split(res.Result().Cookies()[0].Value, ".")

	userUsername := username_password[0]
	userPasswordHash := username_password[1]

	assert.Equal(t, req.Context(), passedContext, "The handler must pass request's context to the service")
	assert.Equal(t, "oussama", userUsername)
	assert.Equal(t, "pass1234", userPasswordHash)
}

func TestErrorHandlingOfSignupHandler(t *testing.T) {
	t.Parallel()
	type SignupTestCase struct {
		description   string
		signupFunc    func(context.Context, string, string) (string, error)
		body          string
		expectedCode  int
		expectedBody  string
		expectedToken string
	}

	exErr := errors.New("example error")

	testCases := []SignupTestCase{
		{
			description:   "Username already exists",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "", domain.ErrDuplicateUsername },
			body:          `{}`,
			expectedCode:  http.StatusConflict,
			expectedBody:  auth.ErrUsernameAlreadyExistsStr,
			expectedToken: "",
		},
		{
			description:   "Invalid username format",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "", auth.ErrInvalidUsernameFormat },
			body:          `{}`,
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidUsernameFormatStr,
			expectedToken: "",
		},
		{
			description:   "weak password",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "", auth.ErrWeakPassword },
			body:          `{}`,
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrWeakPasswordStr,
			expectedToken: "",
		},
		{
			description:   "too long password",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "", auth.ErrPasswordTooLong },
			body:          `{}`,
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrPasswordTooLongStr,
			expectedToken: "",
		},
		{
			description:   "non json request",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "", nil },
			body:          `{`,
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidRequestFormatStr,
			expectedToken: "",
		},
		{
			description: "hashing error",
			signupFunc: func(ctx context.Context, s1, s2 string) (string, error) {
				return "", errors.Join(domain.UnexpectedPasswordHashingError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "database failure",
			signupFunc: func(ctx context.Context, s1, s2 string) (string, error) {
				return "", errors.Join(domain.UnexpectedDatabaseError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "token generation failure",
			signupFunc: func(ctx context.Context, s1, s2 string) (string, error) {
				return "", errors.Join(domain.UnexpectedTokenGenerationError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrAccountCreatedButNoToken,
			expectedToken: "",
		},
		{
			description: "timeout error",
			signupFunc: func(ctx context.Context, s1, s2 string) (string, error) {
				return "", context.DeadlineExceeded
			},
			body:          `{}`,
			expectedCode:  http.StatusGatewayTimeout,
			expectedBody:  auth.ErrServerTimeoutStr,
			expectedToken: "",
		},
		{
			description: "client closed request",
			signupFunc: func(ctx context.Context, s1, s2 string) (string, error) {
				return "", context.Canceled
			},
			body:          `{}`,
			expectedCode:  499,
			expectedBody:  "",
			expectedToken: "",
		},
		{
			description:   "normal",
			signupFunc:    func(ctx context.Context, s1, s2 string) (string, error) { return "tokenhaha", nil },
			body:          `{}`,
			expectedCode:  http.StatusCreated,
			expectedBody:  "",
			expectedToken: "tokenhaha",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			authService := &MockAuthService{}
			authService.SignupFunc = tc.signupFunc
			authHandler := auth.NewAuthHandler(authService, 197*time.Second)

			server := gin.New()

			server.POST("/signup", authHandler.SignupHandler)
			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			cookies := res.Result().Cookies()
			token := ""
			if len(cookies) > 0 {
				assert.Equal(t, "token", cookies[0].Name, "Token cookie must be 'token'")
				assert.Equal(t, "/", cookies[0].Path, "Cookie path must be '/'")
				assert.Equal(t, 197, cookies[0].MaxAge, "Cookie max age mismatch")
				token = cookies[0].Value

			}
			code := res.Code
			body := res.Body.String()

			assert.Equal(t, tc.expectedCode, code, "HTTP status code mismatch")
			assert.Equal(t, tc.expectedBody, body)
			assert.Equal(t, tc.expectedToken, token)
		})
	}
}

// Same reasoning as for TestVariablePassingSignupHandler
func TestVariablePassingLoginHandler(t *testing.T) {
	t.Parallel()
	authService := MockAuthService{}
	var passedContext context.Context

	authService.LoginFunc = func(ctx context.Context, username, password string) (string, error) {
		passedContext = ctx
		return username + "." + password, nil
	}

	authHandler := auth.NewAuthHandler(&authService, 50)

	server := gin.New()
	server.POST("/login", authHandler.LoginHandler)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"oussama", "password":"pass1234"}`))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	username_password := strings.Split(res.Result().Cookies()[0].Value, ".")

	userUsername := username_password[0]
	userPasswordHash := username_password[1]

	assert.Equal(t, req.Context(), passedContext, "The handler must pass request's context to the service")
	assert.Equal(t, "oussama", userUsername)
	assert.Equal(t, "pass1234", userPasswordHash)
}

func TestErrorHandlingOfLoginHandler(t *testing.T) {
	t.Parallel()
	type LoginTestCase struct {
		description   string
		loginFunc     func(context.Context, string, string) (string, error)
		body          string
		expectedCode  int
		expectedBody  string
		expectedToken string
	}

	exErr := errors.New("example error")

	testCases := []LoginTestCase{
		{
			description:   "User not found",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", domain.ErrUserNotFound },
			body:          `{}`,
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  auth.ErrInvalidCredentialsStr,
			expectedToken: "",
		},
		{
			description:   "Incorrect password",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", auth.ErrIncorrectPassword },
			body:          `{}`,
			expectedCode:  http.StatusUnauthorized,
			expectedBody:  auth.ErrInvalidCredentialsStr,
			expectedToken: "",
		},
		{
			description:   "Non json request",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", nil },
			body:          `{`,
			expectedCode:  http.StatusBadRequest,
			expectedBody:  auth.ErrInvalidRequestFormatStr,
			expectedToken: "",
		},
		{
			description:   "Timeout error",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", context.DeadlineExceeded },
			body:          `{}`,
			expectedCode:  http.StatusGatewayTimeout,
			expectedBody:  auth.ErrServerTimeoutStr,
			expectedToken: "",
		},
		{
			description:   "Client closed request",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", context.Canceled },
			body:          `{}`,
			expectedCode:  499,
			expectedBody:  "",
			expectedToken: "",
		},
		{
			description: "Database failure",
			loginFunc: func(ctx context.Context, u, p string) (string, error) {
				return "", errors.Join(domain.UnexpectedDatabaseError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "Hash comparison failure",
			loginFunc: func(ctx context.Context, u, p string) (string, error) {
				return "", errors.Join(domain.UnexpectedPasswordHashComparisonError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description: "Token generation failure",
			loginFunc: func(ctx context.Context, u, p string) (string, error) {
				return "", errors.Join(domain.UnexpectedTokenGenerationError, exErr)
			},
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description:   "Generic unknown error",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "", errors.New("random stuff") },
			body:          `{}`,
			expectedCode:  http.StatusInternalServerError,
			expectedBody:  auth.ErrUnknownStr,
			expectedToken: "",
		},
		{
			description:   "Successful login",
			loginFunc:     func(ctx context.Context, u, p string) (string, error) { return "loginToken123", nil },
			body:          `{}`,
			expectedCode:  http.StatusOK,
			expectedBody:  "",
			expectedToken: "loginToken123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			authService := &MockAuthService{}
			authService.LoginFunc = tc.loginFunc
			authHandler := auth.NewAuthHandler(authService, 100*time.Second)

			server := gin.New()

			server.POST("/login", authHandler.LoginHandler)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			cookies := res.Result().Cookies()
			token := ""
			if len(cookies) > 0 {
				assert.Equal(t, "token", cookies[0].Name, "Token cookie must be 'token'")
				assert.Equal(t, "/", cookies[0].Path, "Cookie path must be '/'")
				assert.Equal(t, 100, cookies[0].MaxAge, "Cookie max age mismatch")
				token = cookies[0].Value
			}
			code := res.Code
			body := res.Body.String()

			assert.Equal(t, tc.expectedCode, code, "HTTP status code mismatch")
			assert.Equal(t, tc.expectedBody, body)
			assert.Equal(t, tc.expectedToken, token)
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	t.Parallel()
	authService := &MockAuthService{}
	authHandler := auth.NewAuthHandler(authService, 4)
	server := gin.New()

	server.POST("/logout", authHandler.LogoutHandler)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	cookies := res.Result().Cookies()

	require.NotEmpty(t, cookies, "Must be a cookie here")
	assert.Equal(t, "token", cookies[0].Name, "deleted cookie name must be 'token'")
	assert.Less(t, cookies[0].MaxAge, 0, "token age must be negative so the cookie gets deleted")
}

func TestRequireAuthMiddleware(t *testing.T) {
	t.Parallel()
	authService := &MockAuthService{}
	authHandler := auth.NewAuthHandler(authService, 15)
	server := gin.New()
	server.Use(authHandler.RequireAuthMiddleware(15 * time.Millisecond))

	passedId := ""

	// Example of not letting non logged users to play
	server.GET("/play", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
		id := ctx.GetString("id")
		passedId = id
	})

	// Case 1, no cookie
	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "Status Code Mismatch")
	assert.Equal(t, auth.ErrMissingTokenStr, res.Body.String())

	// Case 2, Cookie exist and is valid
	authService.VerifyTokenFunc = func(s string) (string, error) { return "id11", nil }
	req = httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "oussama"})
	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "id11", passedId)

	// Case 3, Cookie exist but auth/ package says it's invalid
	authService.VerifyTokenFunc = func(s string) (string, error) { return "", domain.ErrExpiredToken }
	req = httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "oussama"})

	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "Status Code Mismatch")
	assert.Equal(t, auth.ErrExpiredTokenStr, res.Body.String())
}
