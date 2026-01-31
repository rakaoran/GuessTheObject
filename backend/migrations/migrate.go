package migrations

import (
	"database/sql"
	"embed"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *sql
var embedMigrations embed.FS

func Migrate(pgurl string) {
	migrationDB, err := sql.Open("pgx", pgurl)
	if err != nil {
		log.Fatal("Failed to open DB for migrations:", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal("Failed to set goose dialect:", err)
	}

	if err := goose.Up(migrationDB, "."); err != nil {
		log.Fatal("Failed to run up migrations:", err)
	}

	if err := migrationDB.Close(); err != nil {
		log.Fatal("Failed to close migration db connection:", err)
	}
	log.Println("Migrations applied successfully!")
}
