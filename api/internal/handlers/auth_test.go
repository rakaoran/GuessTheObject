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
			body:         `{"username":"haaha", "password": "iems"}`,
			expectedCode: http.StatusConflict,
			expectedBody: `{"error": "username-already-exists"}`,
			cookieIssued: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			authService := &MockAuthService{}
			authService.signupFunc = tc.signupFunc
			authHandler := handlers.NewAuthHandler(authService)
			r := gin.New()

			r.POST("/signup", authHandler.SignupHandler)

			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json;")

			recorder := httptest.NewRecorder()

			r.ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectedCode, recorder.Result().StatusCode)
			assert.JSONEq(t, tc.expectedBody, recorder.Body.String())
			assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json", "Expected json type got %s", recorder.Header().Get("Content-Type"))
			if tc.cookieIssued {
				cookies := recorder.Result().Cookies()
				require.NotEmpty(t, cookies, "Expect a cookie, found zero")
				cookie := cookies[0]
				assert.Equal(t, "token", cookie.Name, "Cookie name must be 'token'")
				assert.Equal(t, "/", cookie.Path, "Cookie Must be for root /")

			}

		})
	}

}
