package store

import (
	"context"
	"database/sql"
	"fmt"
	"errors"

	_"github.com/jackc/pgx/v5/stdlib"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
}

var ErrorNotFound = errors.New("user not found")

func OpenDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE
	)`

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func InsertUser(ctx context.Context, db *sql.DB, user User) (int, error) {
	query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`
	var id int
	if err := db.QueryRowContext(ctx, query, user.Name, user.Email).Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}
	return id, nil
}

func GetUserByID(ctx context.Context, db *sql.DB, id int) (User, error) {
	query := `SELECT id, name, email FROM users WHERE id = $1`
	var user User
	if err := db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrorNotFound
		}
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}