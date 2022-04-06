package storage

import (
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

func New() (*DB, error) {
	host := strings.TrimSpace(os.Getenv("MYSQL_HOST"))
	port := strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	user := strings.TrimSpace(os.Getenv("MYSQL_USER"))
	password := strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	db := strings.TrimSpace(os.Getenv("MYSQL_DB"))

	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		db,
	)

	conn, err := sqlx.Connect("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	migrationSource := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	_, err = migrate.Exec(conn.DB, "mysql", migrationSource, migrate.Up)
	if err != nil {
		return nil, err
	}

	conn = conn.Unsafe()
	conn.SetMaxIdleConns(100)
	conn.SetMaxOpenConns(100)
	conn.SetConnMaxIdleTime(10 * time.Minute)

	chats := &Chats{conn}
	plugins := &Plugins{conn}
	users := &Users{conn}

	return &DB{
		DB:    conn,
		Chats: chats,
		ChatsPlugins: &ChatsPlugins{
			Chats:   chats,
			Plugins: plugins,
			DB:      conn,
		},
		ChatsUsers: &ChatsUsers{
			Chats: chats,
			Users: users,
			DB:    conn,
		},
		Credentials: &Credentials{conn},
		Files:       &Files{conn},
		Plugins:     plugins,
		Users:       users,
	}, nil
}
