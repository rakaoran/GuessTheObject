package handlers_test

import (
	"api/auth"
	"api/internal/handlers"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAuthService struct {
	signupFunc      func(string, string) (string, error)
	loginFunc       func(string, string) (string, error)
	verifyTokenFunc func(string) (string, error)
}

func (mas *MockAuthService) Signup(username, password string) (string, error) {
	return mas.signupFunc(username, password)
}

func (mas *MockAuthService) Login(username, password string) (string, error) {
	return mas.loginFunc(username, password)

}

func (mas *MockAuthService) VerifyToken(token string) (string, error) {
	return mas.verifyTokenFunc(token)
}

func TestSignupHandler(t *testing.T) {
	type SignupTestCase struct {
		description  string
		signupFunc   func(string, string) (string, error)
		body         string
		expectedCode int
		expectedBody string
		cookieIssued bool
	}

	testCases := []SignupTestCase{
		{
			description:  "Username already exists",
			signupFunc:   func(s1, s2 string) (string, error) { return "", auth.UsernameAlreadyExistsErr },
			body:         `{}`,
			expectedCode: http.StatusConflict,
			expectedBody: `{"error": "username-already-exists"}`,
			cookieIssued: false,
		},
		{
			description:  "Invalid username format",
			signupFunc:   func(s1, s2 string) (string, error) { return "", auth.InvalidUsernameFormatErr },
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error": "invalid-username-format"}`,
			cookieIssued: false,
		},
		{
			description:  "weak password",
			signupFunc:   func(s1, s2 string) (string, error) { return "", auth.WeakPasswordErr },
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error": "weak-password"}`,
			cookieIssued: false,
		},
		{
			description:  "non json request",
			signupFunc:   func(s1, s2 string) (string, error) { return "", nil },
			body:         `{`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error": "bad-request-format"}`,
			cookieIssued: false,
		},
		{
			description:  "unknown error",
			signupFunc:   func(s1, s2 string) (string, error) { return "", auth.UnknownErr },
			body:         `{}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error": "unknown-error"}`,
			cookieIssued: false,
		},
		{
			description:  "normal",
			signupFunc:   func(s1, s2 string) (string, error) { return "tokenhaha", nil },
			body:         `{}`,
			expectedCode: http.StatusCreated,
			expectedBody: `{}`,
			cookieIssued: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			authService := &MockAuthService{}
			authService.signupFunc = tc.signupFunc
			authHandler := handlers.NewAuthHandler(authService)

			server := gin.New()

			server.POST("/signup", authHandler.SignupHandler)
			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			cookies := res.Result().Cookies()
			code := res.Code
			body := res.Body.String()

			assert.Equal(t, tc.expectedCode, code, "HTTP status code mismatch")
			assert.JSONEq(t, tc.expectedBody, body)

			if tc.cookieIssued {
				require.NotEmpty(t, cookies, "Expected a cookie, found none")
				assert.Equal(t, "token", cookies[0].Name, "Cookie name must be 'token'")
				assert.Equal(t, "/", cookies[0].Path, "Cookie path must be '/'")
			}

		})
	}

}

func TestLoginHandler(t *testing.T) {
	type LoginTestCase struct {
		description  string
		loginFunc    func(string, string) (string, error)
		body         string
		expectedCode int
		expectedBody string
		cookieIssued bool
	}

	testCases := []LoginTestCase{
		{
			description:  "Username not found",
			loginFunc:    func(s1, s2 string) (string, error) { return "", auth.UsernameNotFoundErr },
			body:         `{}`,
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error": "invalid-credentials"}`,
			cookieIssued: false,
		},
		{
			description:  "incorrect password",
			loginFunc:    func(s1, s2 string) (string, error) { return "", auth.IncorrectPasswordErr },
			body:         `{}`,
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error": "invalid-credentials"}`,
			cookieIssued: false,
		},
		{
			description:  "unknown error",
			loginFunc:    func(s1, s2 string) (string, error) { return "", auth.UnknownErr },
			body:         `{}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error": "unknown-error"}`,
			cookieIssued: false,
		},
		{
			description:  "non json request",
			loginFunc:    func(s1, s2 string) (string, error) { return "", nil },
			body:         `{`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error": "bad-request-format"}`,
			cookieIssued: false,
		},
		{
			description:  "normal",
			loginFunc:    func(s1, s2 string) (string, error) { return "tokenhaha", nil },
			body:         `{}`,
			expectedCode: http.StatusOK,
			expectedBody: `{}`,
			cookieIssued: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			authService := &MockAuthService{}
			authService.loginFunc = tc.loginFunc
			authHandler := handlers.NewAuthHandler(authService)

			server := gin.New()

			server.POST("/login", authHandler.LoginHandler)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			server.ServeHTTP(res, req)

			cookies := res.Result().Cookies()
			code := res.Code
			body := res.Body.String()

			assert.Equal(t, tc.expectedCode, code, "HTTP status code mismatch")
			assert.JSONEq(t, tc.expectedBody, body)

			if tc.cookieIssued {
				require.NotEmpty(t, cookies, "Expected a cookie, found none")
				assert.Equal(t, "token", cookies[0].Name, "Cookie name must be 'token'")
				assert.Equal(t, "/", cookies[0].Path, "Cookie path must be '/'")
			}

		})
	}

}

func TestLogoutHandler(t *testing.T) {
	authService := &MockAuthService{}
	authHandler := handlers.NewAuthHandler(authService)
	server := gin.New()

	server.POST("/logout", authHandler.LogoutHandler)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	cookies := res.Result().Cookies()

	require.NotEmpty(t, cookies, "Must be a cookie here")
	assert.Equal(t, "token", cookies[0].Name, "deleted cookie name must be 'token'")
	assert.Less(t, cookies[0].MaxAge, 0, "token age must be negative so the cookie gets deleted")

	server.ServeHTTP(res, req)
}

func TestRequireAuthMiddleware(t *testing.T) {
	authService := &MockAuthService{}
	authHandler := handlers.NewAuthHandler(authService)
	server := gin.New()
	server.Use(authHandler.RequireAuthMiddleware)

	// Example of not letting non logged users to play
	server.GET("/play", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
		username := ctx.GetString("username")
		assert.Equal(t, "oussama", username, "Token returned a username that the middleware corrupted")
	})

	// Case 1, no cookie
	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "Status Code Mismatch")
	assert.JSONEq(t, `{"error": "missing-token"}`, res.Body.String())

	// Case 2, Cookie exist and is valid
	authService.verifyTokenFunc = func(s string) (string, error) { return s, nil }
	req = httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "oussama"})
	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)

	// Case 3, Cookie exist but auth/ package says it's invalid
	authService.verifyTokenFunc = func(s string) (string, error) { return "", auth.InvalidTokenError }
	req = httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "oussama"})

	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "Status Code Mismatch")
	assert.JSONEq(t, `{"error": "invalid-token"}`, res.Body.String())
}
