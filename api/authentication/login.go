package authentication

import (
	"api/shared/configs"
	"api/shared/database"
	"api/shared/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type loginData struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func loginHandler(ctx *gin.Context) {
	var body loginData

	err := ctx.ShouldBindJSON(&body)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid-format", "message": "You must send 'username' and 'password'"})
		return
	}
	body.Username = strings.ToLower(body.Username)
	userData, err := database.GetUserByUsername(body.Username)

	if err != nil {
		if err.Error() == "user-not-found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user-not-found", "message": "Username not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "server-error"})
		logger.Critical("Server error in loginHandler, user not found for unknown reason")
		return
	}

	correctPassword := verifyPassword(body.Password, userData.PasswordHash)

	if !correctPassword {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect-password", "message": "Entered password is incorrect"})
		return
	}
	ctx.SetSameSite(http.SameSiteNoneMode)
	ctx.SetCookie(configs.JWTCookie.Name,
		getJWT(userData.Id), configs.JWTCookie.MaxAge,
		configs.JWTCookie.Path,
		configs.JWTCookie.Domain,
		configs.JWTCookie.Secure,
		configs.JWTCookie.HttpOnly,
	)
	ctx.JSON(http.StatusCreated, gin.H{"id": userData.Id})
}
