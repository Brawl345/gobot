package storage

import (
	"database/sql"
	"embed"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"time"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

type DB struct {
	*sqlx.DB
	Chats      ChatStorage
	ChatsUsers ChatUserStorage
	Plugins    PluginStorage
	Users      UserStorage
}

func Open(url string) (*DB, error) {
	db, err := sqlx.Open("mysql", url)
	db = db.Unsafe()
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(100)
	db.SetMaxOpenConns(100)
	db.SetConnMaxIdleTime(10 * time.Minute)

	return &DB{
		DB:    db,
		Chats: &Chats{db},
		ChatsUsers: &ChatsUsers{
			Chats: &Chats{db},
			Users: &Users{db},
			DB:    db,
		},
		Plugins: &Plugins{db},
		Users:   &Users{db},
	}, nil
}

func (db *DB) Migrate() (int, error) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	return migrate.Exec(db.DB.DB, "mysql", migrations, migrate.Up)
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
