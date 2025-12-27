package handlers

import (
	"api/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrorMissingTokenJson          = gin.H{"error": "missing-token"}
	ErrorInvalidTokenJson          = gin.H{"error": "invalid-token"}
	ErrorInvalidRequestFormatJson  = gin.H{"error": "bad-request-format"}
	ErrorInvalidCredentialsJson    = gin.H{"error": "invalid-credentials"}
	ErrorUnknownErrorJson          = gin.H{"error": "unknown-error"}
	ErrorUsernameAlreadyExistsJson = gin.H{"error": "username-already-exists"}
	ErrorWeakPasswordJson          = gin.H{"error": "weak-password"}
	ErrorInvalidUsernameFormatJson = gin.H{"error": "invalid-username-format"}
)

type AuthService interface {
	Signup(username, password string) (string, error)
	Login(username, password string) (string, error)
	VerifyToken(token string) (string, error)
}

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{authService: service}
}

func (ah *AuthHandler) RequireAuthMiddleware(ctx *gin.Context) {
	token, err := ctx.Cookie("token")
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorMissingTokenJson)
		return
	}
	username, err := ah.authService.VerifyToken(token)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorInvalidTokenJson)
		return
	}

	ctx.Set("username", username)
	ctx.Next()
}

func (ah *AuthHandler) LoginHandler(ctx *gin.Context) {
	var loginCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := ctx.ShouldBindJSON(&loginCredentials)

	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorInvalidRequestFormatJson)
		return
	}

	token, err := ah.authService.Login(loginCredentials.Username, loginCredentials.Password)

	if err != nil {
		switch err {
		case auth.UsernameNotFoundErr:
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorInvalidCredentialsJson)
		case auth.IncorrectPasswordErr:
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorInvalidCredentialsJson)
		default:
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrorUnknownErrorJson)
		}
		return
	}

	ctx.SetCookie("token", token, 60*60*24*365*40, "/", "", true, true)
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.JSON(http.StatusOK, gin.H{})
}

func (ah *AuthHandler) SignupHandler(ctx *gin.Context) {
	var signupCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := ctx.ShouldBindJSON(&signupCredentials)

	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorInvalidRequestFormatJson)
		return
	}

	token, err := ah.authService.Signup(signupCredentials.Username, signupCredentials.Password)

	if err != nil {
		switch err {
		case auth.UsernameAlreadyExistsErr:
			ctx.AbortWithStatusJSON(http.StatusConflict, ErrorUsernameAlreadyExistsJson)
		case auth.WeakPasswordErr:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorWeakPasswordJson)
		case auth.InvalidUsernameFormatErr:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorInvalidUsernameFormatJson)
		default:
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrorUnknownErrorJson)
		}
		return
	}

	ctx.SetCookie("token", token, 60*60*24*365*40, "/", "", true, true)
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.JSON(http.StatusCreated, gin.H{})
}

func (ah *AuthHandler) LogoutHandler(ctx *gin.Context) {
	ctx.SetCookie("token", "", -1, "/", "", true, true)
}
