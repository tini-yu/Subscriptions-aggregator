package db

import (
	"database/sql"
	"fmt"
	"os"
	"log"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ConnectDB() (*sql.DB, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	host := os.Getenv("host")
	port := os.Getenv("dbport")
	user := os.Getenv("user")
	password := os.Getenv("password")
	dbname := os.Getenv("dbname")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	// Миграция БД:
	migrationURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
	user, password, host, port, dbname)

	m, err := migrate.New("file://database/migrations", migrationURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	return db, nil
}
