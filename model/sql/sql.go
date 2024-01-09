package sql

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

func New() (*sqlx.DB, error) {
	host := strings.TrimSpace(os.Getenv("MYSQL_HOST"))
	if host == "" {
		host = "localhost"
	}
	port := strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	if port == "" {
		port = "3306"
	}
	user := strings.TrimSpace(os.Getenv("MYSQL_USER"))
	password := strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	dbname := strings.TrimSpace(os.Getenv("MYSQL_DB"))
	tls := strings.TrimSpace(os.Getenv("MYSQL_TLS"))
	if tls == "" {
		tls = "false"
	}

	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		user,
		password,
		host,
		port,
		dbname,
		tls,
	)

	db, err := sqlx.Connect("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	_, ignoreMigration := os.LookupEnv("IGNORE_SQL_MIGRATION")
	if !ignoreMigration {
		migrationSource := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
		_, err = migrate.Exec(db.DB, "mysql", migrationSource, migrate.Up)
		if err != nil {
			return nil, err
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
