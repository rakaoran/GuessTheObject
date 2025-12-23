package database

import (
	"api/shared/logger"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type UserData struct {
	Id           string
	Username     string
	PasswordHash string
}

func CreateUser(username string, passwordHash string) (string, error) {
	row := pool.QueryRow(ctx, "INSERT INTO users(username, password_hash) VALUES($1, $2) RETURNING id;", username, passwordHash)
	var id string
	err := row.Scan(&id)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				return "", errors.New("username-already-exists")
			}
			if pgErr.Code == "23514" {
				return "", errors.New("invalid-username-format")
			}
			fmt.Println(pgErr.Code, pgErr.ColumnName, pgErr.ConstraintName)
		}
		logger.Criticalf("Couldn't create user for uncaught exception: %v\n", err)
	}
	return id, nil
}

func GetUserById(id string) (UserData, error) {
	row := pool.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", id)
	var userData UserData
	err := row.Scan(&userData.Id, &userData.Username, &userData.PasswordHash)
	if err != nil {
		return UserData{}, errors.New("user-not-found")
	}
	return userData, nil
}

func GetUserByUsername(username string) (UserData, error) {
	row := pool.QueryRow(ctx, "SELECT * FROM users WHERE username = $1", username)
	var userData UserData
	err := row.Scan(&userData.Id, &userData.Username, &userData.PasswordHash)
	if err != nil {
		return UserData{}, errors.New("user-not-found")
	}
	return userData, nil
}
