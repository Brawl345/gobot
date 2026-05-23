package sql

import (
	"cmp"
	"database/sql"
	"embed"
	"os"
	"time"

	"github.com/Brawl345/gobot/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

var log = logger.New("db")

//go:embed migrations/*
var embeddedMigrations embed.FS

func New() (*sqlx.DB, error) {
	socket := os.Getenv("MYSQL_SOCKET")

	cfg := mysqlDriver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.DBName = os.Getenv("MYSQL_DB")
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Collation = "utf8mb4_unicode_ci"
	cfg.RejectReadOnly = true

	if socket != "" {
		cfg.Net = "unix"
		cfg.Addr = socket
	} else {
		cfg.Net = "tcp"
		cfg.Addr = cmp.Or(os.Getenv("MYSQL_HOST"), "localhost") + ":" + cmp.Or(os.Getenv("MYSQL_PORT"), "3306")
		cfg.TLSConfig = cmp.Or(os.Getenv("MYSQL_TLS"), "false")
	}

	connector, err := mysqlDriver.NewConnector(cfg)
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(sql.OpenDB(connector), "mysql")
	if err := db.Ping(); err != nil {
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
