package authentication

import (
	"api/shared/configs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func logoutHandler(ctx *gin.Context) {
	ctx.SetCookie("token", "", -1, "/", configs.JWTCookie.Domain, true, true)
	ctx.JSON(http.StatusOK, nil)
}
