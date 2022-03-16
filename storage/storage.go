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
	Chats        ChatStorage
	ChatsPlugins ChatPluginStorage
	ChatsUsers   ChatUserStorage
	Plugins      PluginStorage
	Users        UserStorage
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

	chats := &Chats{db}
	plugins := &Plugins{db}
	users := &Users{db}

	return &DB{
		DB:    db,
		Chats: chats,
		ChatsPlugins: &ChatsPlugins{
			Chats:   chats,
			Plugins: plugins,
			DB:      db,
		},
		ChatsUsers: &ChatsUsers{
			Chats: chats,
			Users: users,
			DB:    db,
		},
		Plugins: plugins,
		Users:   users,
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
