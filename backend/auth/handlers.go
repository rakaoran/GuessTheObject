package auth

import (
	"api/domain"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

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
			clientIP := ctx.ClientIP()
			userAgent := ctx.Request.UserAgent()
			tokenParts := strings.Split(token, ".")
			redactedToken := ""
			if len(tokenParts) == 3 {
				sneak := ""
				r := []rune(tokenParts[2])

				if len(r) >= 10 {
					sneak = string(r[:10]) + strings.Repeat("*", len(r)-10)
				} else {
					sneak = tokenParts[2]
				}
				redactedToken = tokenParts[0] + "." + tokenParts[1] + "." + sneak
			} else {
				redactedToken = token
			}

			switch {
			case errors.Is(err, domain.ErrInvalidSigningAlg),
				errors.Is(err, domain.ErrInvalidTokenSignature),
				errors.Is(err, domain.ErrCorruptedToken):

				slog.Warn("RequireAuthMiddleware: suspicious token attempt",
					"ip", clientIP,
					"user_agent", userAgent,
					"error", err.Error(),
					"token", redactedToken,
				)

				time.Sleep(trollTime)
				ctx.Status(http.StatusInternalServerError)
				ctx.Abort()

			case errors.Is(err, domain.ErrExpiredToken):
				slog.Info("RequireAuthMiddleware: token expired", "ip", clientIP, "token", redactedToken)
				ctx.String(http.StatusUnauthorized, ErrExpiredTokenStr)
				ctx.Abort()

			default:

				slog.Error("RequireAuthMiddleware: internal auth error",
					"ip", clientIP,
					"error", err.Error(),
					"token", redactedToken,
				)

				ctx.String(http.StatusUnauthorized, ErrUnknownStr)
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
		clientIP := ctx.ClientIP()
		userAgent := ctx.Request.UserAgent()
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

		case errors.Is(err, domain.UnexpectedDatabaseError):
			slog.Error("Login: Database returned an unexpected error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", loginCredentials.Username,
			)
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()

		case errors.Is(err, domain.UnexpectedPasswordHashComparisonError):
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			slog.Error("Login: Hashing comparison error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", loginCredentials.Username,
				"password_len", utf8.RuneCountInString(loginCredentials.Password),
				"mem_alloc_mb", (mem.Alloc/1024)/1024,
				"mem_sys_mb", (mem.Sys/1024)/1024,
			)
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()

		case errors.Is(err, domain.UnexpectedTokenGenerationError):
			slog.Error("Login: Token generation error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", loginCredentials.Username,
			)
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)
			ctx.Abort()
		default:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			slog.Error("Login: Unknown unexpected error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", loginCredentials.Username,
				"password_len", utf8.RuneCountInString(loginCredentials.Password),
				"mem_alloc_mb", (mem.Alloc/1024)/1024,
				"mem_sys_mb", (mem.Sys/1024)/1024,
			)
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
		clientIP := ctx.ClientIP()
		userAgent := ctx.Request.UserAgent()

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
			ctx.Status(499)

		case errors.Is(err, domain.UnexpectedDatabaseError):
			slog.Error("Signup: Database returned an unexpected error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", signupCredentials.Username,
			)
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)

		case errors.Is(err, domain.UnexpectedPasswordHashingError):
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			slog.Error("Signup: Password hashing error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", signupCredentials.Username,
				"password_len", utf8.RuneCountInString(signupCredentials.Password),
				"mem_alloc_mb", (mem.Alloc/1024)/1024,
				"mem_sys_mb", (mem.Sys/1024)/1024,
			)
			ctx.String(http.StatusInternalServerError, ErrUnknownStr)

		case errors.Is(err, domain.UnexpectedTokenGenerationError):
			slog.Error("Signup: Token generation error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", signupCredentials.Username,
			)
			ctx.String(http.StatusInternalServerError, ErrAccountCreatedButNoToken)

		default:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			slog.Error("Signup: Unknown unexpected error",
				"error", err.Error(),
				"ip", clientIP,
				"user_agent", userAgent,
				"username", signupCredentials.Username,
				"password_len", utf8.RuneCountInString(signupCredentials.Password),
				"mem_alloc_mb", (mem.Alloc/1024)/1024,
				"mem_sys_mb", (mem.Sys/1024)/1024,
			)
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
		clientIP := ctx.ClientIP()
		userAgent := ctx.Request.UserAgent()
		tokenParts := strings.Split(token, ".")
		redactedToken := ""
		if len(tokenParts) == 3 {
			sneak := ""
			r := []rune(tokenParts[2])
			if len(r) >= 10 {
				sneak = string(r[:10]) + strings.Repeat("*", len(r)-10)
			} else {
				sneak = tokenParts[2]
			}
			redactedToken = tokenParts[0] + "." + tokenParts[1] + "." + sneak
		} else {
			redactedToken = token
		}

		slog.Warn("Refresh: Invalid token provided",
			"ip", clientIP,
			"user_agent", userAgent,
			"error", err.Error(),
			"token", redactedToken,
		)
		ctx.String(http.StatusUnauthorized, "bad-token")
		return
	}

	newToken, err := ah.authService.GenerateToken(id)
	if err != nil {
		clientIP := ctx.ClientIP()
		userAgent := ctx.Request.UserAgent()
		slog.Error("Refresh: Failed to generate new token",
			"ip", clientIP,
			"user_agent", userAgent,
			"error", err.Error(),
			"user_id", id,
		)
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
