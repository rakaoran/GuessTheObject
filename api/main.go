package main

import (
	"api/auth"
	"api/crypto"
	"api/database"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CreateServer(allowedOrigins []string) *gin.Engine {
	r := gin.New()
	r.GET("/health", func(ctx *gin.Context) { ctx.String(200, "healthy") })

	r.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		println(origin)
		if slices.Contains(allowedOrigins, origin) {
			ctx.Next()
			return
		}
		ctx.String(http.StatusForbidden, "forbidden origin")
		ctx.Abort()
	})

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	return r
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	// ENVs
	ALLOWED_ORIGINS, exists := os.LookupEnv("ALLOWED_ORIGINS")
	if !exists {
		log.Fatal("Missing allowed origins")
	}
	allowedOrigins := strings.Split(ALLOWED_ORIGINS, ",")
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
	tokenAge := time.Hour * 24 * 7 // 7 days
	passwordHasher := crypto.NewArgon2idHasher(3, 1024*64, 32, 16, 1)
	tokenManager := crypto.NewJWTManager(JWT_KEY, tokenAge)

	authService := auth.NewService(pgRepo, passwordHasher, tokenManager)
	authHandler := auth.NewAuthHandler(authService, tokenAge)

	r := CreateServer(allowedOrigins)

	// ! WARNING, make sure the Routes are correctly binded as the bindings are not tested
	{
		auth := r.Group("/auth")
		auth.POST("/signup", authHandler.SignupHandler)
		auth.POST("/login", authHandler.LoginHandler)
		auth.POST("/logout", authHandler.LogoutHandler)
		auth.GET("/refresh", authHandler.RefreshSessionHandler)
	}

	r.Run(":5000")
}
