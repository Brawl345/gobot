package storage

import (
	"embed"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

type DB struct {
	*sqlx.DB
	Chats        ChatStorage
	ChatsPlugins ChatPluginStorage
	ChatsUsers   ChatUserStorage
	Credentials  CredentialStorage
	Files        FileStorage
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
		Credentials: &Credentials{db},
		Files:       &Files{db},
		Plugins:     plugins,
		Users:       users,
	}, nil
}

func (db *DB) Migrate() (int, error) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	return migrate.Exec(db.DB.DB, "mysql", migrations, migrate.Up)
}
