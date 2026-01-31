package main

import (
	"api/auth"
	"api/crypto"
	"api/game"
	"api/storage"
	"context"
	"database/sql"
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func CreateServer(allowedOrigins []string) *gin.Engine {
	r := gin.New()
	r.GET("/health", func(ctx *gin.Context) { ctx.String(200, "healthy") })

	r.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")

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
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Content-Type",
			"Authorization",
			"Upgrade",
			"Connection",
			"Sec-WebSocket-Key",
			"Sec-WebSocket-Version",
			"Sec-WebSocket-Extensions",
			"Sec-WebSocket-Protocol",
		},
	}))

	return r
}

//go:embed migrations/*sql
var embedMigrations embed.FS

func main() {

	// logger setup
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

	// ---------------------------
	// 🦆 GOOSE MIGRATIONS START
	// ---------------------------
	migrationDB, err := sql.Open("pgx", POSTGRES_URL)
	if err != nil {
		log.Fatal("Failed to open DB for migrations:", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal("Failed to set goose dialect:", err)
	}

	if err := goose.Up(migrationDB, "migrations"); err != nil {
		log.Fatal("Failed to run up migrations:", err)
	}

	if err := migrationDB.Close(); err != nil {
		log.Fatal("Failed to close migration db connection:", err)
	}
	log.Println("Migrations applied successfully!")
	// ---------------------------
	// 🦆 GOOSE MIGRATIONS END
	// ---------------------------

	// Dependencies
	pgRepo, err := storage.NewPostgresRepo(context.Background(), POSTGRES_URL)
	if err != nil {
		log.Fatal(err)
	}
	tokenAge := time.Hour * 24 * 7 // 7 days
	passwordHasher := crypto.NewArgon2idHasher(3, 1024*64, 32, 16, 1)
	tokenManager := crypto.NewJWTManager(JWT_KEY, tokenAge)

	authService := auth.NewService(pgRepo, passwordHasher, tokenManager)
	authHandler := auth.NewAuthHandler(authService, tokenAge)

	r := CreateServer(allowedOrigins)

	{
		auth := r.Group("/auth")
		auth.POST("/signup", authHandler.SignupHandler)
		auth.POST("/login", authHandler.LoginHandler)
		auth.POST("/logout", authHandler.LogoutHandler)
		auth.GET("/refresh", authHandler.RefreshSessionHandler)
	}

	idGen := game.NewIdGen()
	tickerGen := game.NewTickerGen()
	wg := sync.WaitGroup{}
	lobby := game.NewLobby(&idGen, &tickerGen, &wg)

	lobbyStarted := make(chan struct{})
	go lobby.LobbyActor(lobbyStarted)
	<-lobbyStarted

	gameHandler := game.NewGameHandler(lobby, pgRepo, pgRepo)
	{
		gameGroup := r.Group("/game")
		gameGroup.Use(authHandler.RequireAuthMiddleware(time.Second * 2))

		gameGroup.GET("/create", gameHandler.CreateGameHandler)

		gameGroup.GET("/join/:roomid", gameHandler.JoinGameHandler)
		gameGroup.GET("/games", gameHandler.GetPublicGamesHandler)
	}

	go r.Run(":5000")
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
	println("Server started")
	<-sigCh
	println("SIGTERM or SIGINT received, waiting for rooms to finish before shutting down")

	wg.Wait()
	println("Shutting down now")

}
