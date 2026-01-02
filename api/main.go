package main

import (
	"api/auth"
	"api/crypto"
	"api/database"
	"context"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"status": "ok"}) })

	// ENVs
	ALLOWED_ORIGINS, exists := os.LookupEnv("ALLOWED_ORIGINS")
	if !exists {
		log.Fatal("Missing allowed origins")
	}
	allowedOriginsArray := strings.Split(ALLOWED_ORIGINS, ",")
	POSTGRES_URL, exists := os.LookupEnv("POSTGRES_URL")
	if !exists {
		log.Fatal("Missing postgres url")
	}
	JWT_KEY, exists := os.LookupEnv("JWT_KEY")
	if !exists {
		log.Fatal("Missing jwt signing key")
	}
	// Dependencies
	pgRepo, err := database.NewPostgresRepo(context.Background(), POSTGRES_URL)
	if err != nil {
		log.Fatal(err)
	}
	passwordHasher := crypto.NewArgon2idHasher(3, 1024*64, 32, 16, 1)
	tokenManager := crypto.NewJWTManager(JWT_KEY, 60*60*24*7)

	authService := auth.NewService(pgRepo, passwordHasher, tokenManager)
	authHandler := auth.NewAuthHandler(authService)

	r.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		println(origin)
		if slices.Contains(allowedOriginsArray, origin) {
			println("yes")
			ctx.Next()
			return
		}
		println("no")
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "non authorized origin"})
	})
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOriginsArray,
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type"},
		AllowMethods:     []string{"GET", "POST"},
	}))
	r.POST("/signup", authHandler.SignupHandler)
	r.POST("/login", authHandler.LoginHandler)

	r.Run(":5000")
}
