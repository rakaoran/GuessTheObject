package authentication

import "github.com/gin-gonic/gin"

func RegisterRoute(engine *gin.Engine) {
	engine.POST("/authentication/signup", signupHandler)
	engine.POST("/authentication/login", loginHandler)
	engine.POST("/authentication/logout", logoutHandler)
}
