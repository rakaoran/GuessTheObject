package main

import (
	"api/internal/authentication"
	"api/internal/game"
	"api/internal/shared/configs"
	"api/internal/shared/database"
	"api/internal/shared/logger"
	"slices"

	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	database.Initialize()
	game.LoadWords()
	game.StartTickers()

	var allowedOrigins = []string{}
	println(configs.Envs.GIN_MODE, "meowwwwwwwww")
	if configs.Envs.GIN_MODE == "release" {
		allowedOrigins = append(allowedOrigins, "https://"+configs.Envs.FRONTEND_ORIGIN)
		allowedOrigins = append(allowedOrigins, "https://www."+configs.Envs.FRONTEND_ORIGIN)
	} else {
		allowedOrigins = append(allowedOrigins, "http://"+configs.Envs.FRONTEND_ORIGIN)
	}

	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		if slices.Contains(allowedOrigins, origin) {
			ctx.Next()
			return
		}

		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "forbidden origin",
		})
		ctx.Abort()
	})

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type", "Origin"},
	}))

	authentication.RegisterRoute(r)
	game.RegisterRoute(r)

	fmt.Println("api listening on port 5000")
	err := r.Run(":5000")
	logger.Fatalf("Couldn't start server: %v", err)
}
