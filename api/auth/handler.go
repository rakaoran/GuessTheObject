package auth

import (
	"api/domain"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrMissingTokenStr          = "missing-token"
	ErrExpiredTokenStr          = "expired-token"
	ErrServerTimeoutStr         = "server-timeout"
	ErrInvalidRequestFormatStr  = "bad-request-format"
	ErrInvalidCredentialsStr    = "invalid-credentials"
	ErrUnknownStr               = "unknown-error"
	ErrUsernameAlreadyExistsStr = "username-already-exists"
	ErrWeakPasswordStr          = "weak-password"
	ErrPasswordTooLongStr       = "password-too-long"
	ErrInvalidUsernameFormatStr = "invalid-username-format"
	ErrAccountCreatedButNoToken = "account-created-but-no-token"
)

type authHandler struct {
	authService  AuthService
	cookieMaxAge time.Duration
}

func NewAuthHandler(service AuthService, cookieMaxAge time.Duration) *authHandler {
	return &authHandler{authService: service, cookieMaxAge: cookieMaxAge}
}

func (ah *authHandler) RequireAuthMiddleware(trollTime time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie("token")
		if err != nil {
			ctx.String(http.StatusUnauthorized, ErrMissingTokenStr)
			ctx.Abort()
			return
		}
		id, err := ah.authService.VerifyToken(token)

		if err != nil {
			// TODO: log what is happening
			switch {
			case errors.Is(err, domain.ErrInvalidSigningAlg), errors.Is(err, domain.ErrInvalidTokenSignature), errors.Is(err, domain.ErrCorruptedToken):
				time.Sleep(trollTime)
				ctx.Status(http.StatusInternalServerError)
				ctx.Abort()
			case errors.Is(err, domain.ErrExpiredToken):
				ctx.String(http.StatusUnauthorized, ErrExpiredTokenStr)
				ctx.Abort()
			default:
				ctx.String(http.StatusInternalServerError, ErrUnknownStr)
				ctx.Abort()
			}

			return
		}

		ctx.Set("id", id)
		ctx.Next()
	}
}

func (ah *authHandler) LoginHandler(ctx *gin.Context) {
	var loginCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := ctx.ShouldBindJSON(&loginCredentials)

	if err != nil {
		ctx.String(http.StatusBadRequest, ErrInvalidRequestFormatStr)
		ctx.Abort()
		return
	}

	reqCtx := ctx.Request.Context()

	token, err := ah.authService.Login(reqCtx, loginCredentials.Username, loginCredentials.Password)

	if err != nil {
		switch {
		case errors.Is(err, ErrIncorrectPassword), errors.Is(err, domain.ErrUserNotFound):
			ctx.String(http.StatusUnauthorized, ErrInvalidCredentialsStr)
			ctx.Abort()
		case errors.Is(err, context.DeadlineExceeded):
			ctx.String(http.StatusGatewayTimeout, ErrServerTimeoutStr)
			ctx.Abort()
		case errors.Is(err, context.Canceled):
			ctx.Status(499)
			ctx.Abort()

		// TODO: for each case, log what happened with necessary info to help identify what caused the unexpected error
		case errors.Is(err, domain.UnexpectedDatabaseError):
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()
		case errors.Is(err, domain.UnexpectedPasswordHashComparisonError):
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()
		case errors.Is(err, domain.UnexpectedTokenGenerationError):
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()
		default:
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()
		}
		return
	}

	ctx.SetCookie("token", token, int(ah.cookieMaxAge.Seconds()), "/", "", true, true)
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.Status(http.StatusOK)
}

func (ah *authHandler) SignupHandler(ctx *gin.Context) {
	var signupCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := ctx.ShouldBindJSON(&signupCredentials)

	if err != nil {
		ctx.String(http.StatusBadRequest, ErrInvalidRequestFormatStr)
		ctx.Abort()
		return
	}

	reqCtx := ctx.Request.Context()

	token, err := ah.authService.Signup(reqCtx, signupCredentials.Username, signupCredentials.Password)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDuplicateUsername):
			ctx.String(http.StatusConflict, ErrUsernameAlreadyExistsStr)

		case errors.Is(err, ErrWeakPassword):
			ctx.String(http.StatusBadRequest, ErrWeakPasswordStr)

		case errors.Is(err, ErrPasswordTooLong):
			ctx.String(http.StatusBadRequest, ErrPasswordTooLongStr)

		case errors.Is(err, ErrInvalidUsernameFormat):
			ctx.String(http.StatusBadRequest, ErrInvalidUsernameFormatStr)

		case errors.Is(err, context.DeadlineExceeded):
			ctx.String(http.StatusGatewayTimeout, ErrServerTimeoutStr)

		case errors.Is(err, context.Canceled):
			ctx.Status(499) // http code for "Client Closed Request"

		// TODO: for each case, log what happened with necessary info to help identify what caused the unexpected error
		case errors.Is(err, domain.UnexpectedDatabaseError):
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)

		case errors.Is(err, domain.UnexpectedPasswordHashComparisonError):
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)

		case errors.Is(err, domain.UnexpectedTokenGenerationError):
			ctx.String(http.StatusInternalServerError, ErrAccountCreatedButNoToken)

		default:
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
		}
		ctx.Abort()
		return
	}

	ctx.SetCookie("token", token, int(ah.cookieMaxAge.Seconds()), "/", "", true, true)
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.Status(http.StatusCreated)
}

func (ah *authHandler) RefreshSessionHandler(ctx *gin.Context) {
	token, err := ctx.Cookie("token")
	if err != nil {
		ctx.String(http.StatusUnauthorized, "unauthenticated")
		return
	}

	id, err := ah.authService.VerifyToken(token)
	if err != nil {
		ctx.String(http.StatusUnauthorized, "bad-token")
		return
	}

	newToken, err := ah.authService.GenerateToken(id)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.SetCookie("token", newToken, int(ah.cookieMaxAge.Seconds()), "/", "", true, true)
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.Status(http.StatusOK)
}

func (ah *authHandler) LogoutHandler(ctx *gin.Context) {
	ctx.SetCookie("token", "", -1, "/", "", true, true)
}
