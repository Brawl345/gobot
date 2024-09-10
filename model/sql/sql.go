package sql

import (
	"cmp"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

var log = logger.New("db")

//go:embed migrations/*
var embeddedMigrations embed.FS

func New() (*sqlx.DB, error) {
	host := cmp.Or(strings.TrimSpace(os.Getenv("MYSQL_HOST")), "localhost")
	port := cmp.Or(strings.TrimSpace(os.Getenv("MYSQL_PORT")), "3306")
	user := strings.TrimSpace(os.Getenv("MYSQL_USER"))
	password := strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	dbname := strings.TrimSpace(os.Getenv("MYSQL_DB"))
	tls := cmp.Or(strings.TrimSpace(os.Getenv("MYSQL_TLS")), "false")
	socket := strings.TrimSpace(os.Getenv("MYSQL_SOCKET"))

	var connectionString string
	if socket != "" {
		connectionString = fmt.Sprintf(
			"%s@unix(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user,
			socket,
			dbname,
		)
	} else {
		connectionString = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			user,
			password,
			host,
			port,
			dbname,
			tls,
		)
	}

	db, err := sqlx.Connect("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	_, ignoreMigration := os.LookupEnv("IGNORE_SQL_MIGRATION")
	if !ignoreMigration {
		migrationSource := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
		applied, err := migrate.Exec(db.DB, "mysql", migrationSource, migrate.Up)
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

	log.Debug().Msgf("Connected to database")

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
