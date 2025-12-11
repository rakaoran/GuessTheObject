package game

import "github.com/gin-gonic/gin"

func RegisterRoute(engine *gin.Engine) {
	engine.GET("/matchmaking", matchmakingHandler)
	engine.GET("/join/:gameid", joinByLink)
}
