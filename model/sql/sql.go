package sql

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
)

var log = logger.New("db")

//go:embed migrations/*
var embeddedMigrations embed.FS

func New() (*sqlx.DB, error) {
	host := strings.TrimSpace(os.Getenv("POSTGRESQL_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("POSTGRESQL_PORT"))
	if port == "" {
		port = "5432"
	}
	user := strings.TrimSpace(os.Getenv("POSTGRESQL_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRESQL_PASSWORD"))
	dbname := strings.TrimSpace(os.Getenv("POSTGRESQL_DB"))
	sslmode := strings.TrimSpace(os.Getenv("POSTGRESQL_SSLMODE"))
	if sslmode == "" {
		sslmode = "disable"
	}

	connectionString := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		user,
		password,
		host,
		port,
		dbname,
		sslmode,
	)

	db, err := sqlx.Connect("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	_, ignoreMigration := os.LookupEnv("IGNORE_SQL_MIGRATION")
	if !ignoreMigration {
		migrationSource := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
		applied, err := migrate.Exec(db.DB, "postgres", migrationSource, migrate.Up)
		if err != nil {
			return nil, err
		}
		if applied != 0 {
			log.Info().Msgf("Applied %d migrations", applied)
		}
	}

	db = db.Unsafe()
	db.SetMaxIdleConns(100)
	db.SetMaxOpenConns(100)
	db.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}

func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
