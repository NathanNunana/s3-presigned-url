package store

import (
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

func Connect() (*surrealdb.DB, error) {
	db, err := surrealdb.New("ws://localhost:8001")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SurrealDB: %w", err)
	}

	if err := db.Use("demo", "demo"); err != nil {
		return nil, fmt.Errorf("failed to select namespace and database: %w", err)
	}

	authData := &surrealdb.Auth{
		Username: "root",
		Password: "slurp",
	}
	token, err := db.SignIn(authData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	if err := db.Authenticate(token); err != nil {
		return nil, fmt.Errorf("failed to authenticate with token: %w", err)
	}

	return db, nil
}

func Disconnect(db *surrealdb.DB) error {
	if err := db.Invalidate(); err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}
	return db.Close()
}
