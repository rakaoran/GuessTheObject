package auth

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// var (
// 	ErrorMissingTokenJson          = "missing-token"
// 	ErrorExpiredTokenJson          = "expired-token"
// 	ErrorInvalidTokenJson          = "invalid-token"
// 	ErrorInvalidRequestFormatJson  = "bad-request-format"
// 	ErrorInvalidCredentialsJson    = "invalid-credentials"
// 	ErrorUnknownErrorJson          = "unknown-error"
// 	ErrorUsernameAlreadyExistsJson = "username-already-exists"
// 	ErrorWeakPasswordJson          = "weak-password"
// 	ErrorInvalidUsernameFormatJson = "invalid-username-format"
// )

// type AuthHandler struct {
// 	authService *AuthService
// }

// func NewAuthHandler(service *AuthService) *AuthHandler {
// 	return &AuthHandler{authService: service}
// }

// func (ah *AuthHandler) RequireAuthMiddleware(ctx *gin.Context) {
// 	token, err := ctx.Cookie("token")
// 	if err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorMissingTokenJson)
// 		return
// 	}
// 	id, err := ah.authService.VerifyToken(token)
// 	if err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorInvalidTokenJson)
// 		return
// 	}

// 	ctx.Set("id", id)
// 	ctx.Next()
// }

// func (ah *AuthHandler) LoginHandler(ctx *gin.Context) {
// 	var loginCredentials struct {
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	}

// 	err := ctx.ShouldBindJSON(&loginCredentials)

// 	if err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorInvalidRequestFormatJson)
// 		return
// 	}

// 	token, err := ah.authService.Login(ctx.Request.Context(), loginCredentials.Username, loginCredentials.Password)

// 	if err != nil {
// 		// TODO: ctx.String()
// 		switch err {
// 		case ErrUsernameNotFound:
// 			ctx.AbortWithStatus(http.StatusUnauthorized)
// 		case ErrIncorrectPassword:
// 			ctx.AbortWithStatus(http.StatusUnauthorized)
// 		default:
// 			ctx.AbortWithStatus(http.StatusInternalServerError)
// 		}
// 		return
// 	}

// 	ctx.SetCookie("token", token, 60*60*24*365*40, "/", "", true, true)
// 	ctx.SetSameSite(http.SameSiteNoneMode)
// 	ctx.Status(http.StatusOK)
// }

// func (ah *AuthHandler) SignupHandler(ctx *gin.Context) {
// 	var signupCredentials struct {
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	}

// 	err := ctx.ShouldBindJSON(&signupCredentials)

// 	if err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorInvalidRequestFormatJson)
// 		return
// 	}

// 	token, err := ah.authService.Signup(ctx.Request.Context(), signupCredentials.Username, signupCredentials.Password)

// 	if err != nil {
// 		// TODO: ctx.String()
// 		switch err {
// 		case ErrUsernameAlreadyExists:
// 			ctx.AbortWithStatus(http.StatusConflict)
// 		case ErrWeakPassword:
// 			ctx.AbortWithStatus(http.StatusBadRequest)
// 		case ErrInvalidUsernameFormat:
// 			ctx.AbortWithStatus(http.StatusBadRequest)
// 		default:
// 			ctx.AbortWithStatus(http.StatusInternalServerError)
// 		}
// 		return
// 	}

// 	ctx.SetCookie("token", token, 60*60*24*365*40, "/", "", true, true)
// 	ctx.SetSameSite(http.SameSiteNoneMode)
// 	ctx.JSON(http.StatusCreated, gin.H{})
// }

// func (ah *AuthHandler) LogoutHandler(ctx *gin.Context) {
// 	ctx.SetCookie("token", "", -1, "/", "", true, true)
// }
