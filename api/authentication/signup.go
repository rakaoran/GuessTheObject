package authentication

import (
	"api/shared/configs"
	"api/shared/database"
	"api/shared/logger"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type signupData struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func signupHandler(ctx *gin.Context) {
	var body signupData
	err := ctx.ShouldBindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid-format", "message": "You must send 'username' and 'password'"})
		return
	}
	body.Username = strings.ToLower(body.Username)
	if matched, _ := regexp.MatchString("^[a-z0-9_]{3,20}$", body.Username); !matched {

		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid-username", "message": "Username must contain only alphabets, numbers, underscores and be between 3 and 20 characters"})
		return
	}

	if len(body.Password) < 8 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "weak-password", "message": "Password should be at least 8 characters"})
		return
	}

	id, err := database.CreateUser(body.Username, hashPassword(body.Password))

	if err != nil {
		if err.Error() == "username-already-exists" {
			ctx.JSON(http.StatusConflict, gin.H{"error": "username-already-exists", "message": "This username is already taken"})
			return
		}
		if err.Error() == "invalid-username-format" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid-username", "message": "Username must contain only alphabets, numbers, underscores and be between 3 and 20 characters"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "server-error"})
		logger.Critical("Server error in singupHandler, user not created for unknown reason")
		return
	}
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.SetCookie(configs.JWTCookie.Name,
		getJWT(id), configs.JWTCookie.MaxAge,
		configs.JWTCookie.Path,
		configs.JWTCookie.Domain,
		configs.JWTCookie.Secure,
		configs.JWTCookie.HttpOnly,
	)
	ctx.JSON(http.StatusCreated, gin.H{"id": id})
}
