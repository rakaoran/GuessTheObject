package configs

import "os"

var Envs = struct {
	FRONTEND_ORIGIN string
	JWT_KEY         []byte
	POSTGRES_URL    string
	GIN_MODE        string
}{
	FRONTEND_ORIGIN: os.Getenv("FRONTEND_ORIGIN"),
	JWT_KEY:         []byte(os.Getenv("JWT_KEY")),
	POSTGRES_URL:    os.Getenv("POSTGRES_URL"),
	GIN_MODE:        os.Getenv("GIN_MODE"),
}
