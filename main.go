package main

import (
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/storage"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/telebot.v3"
	"log"
	"os"
	"runtime/debug"
	"time"
)

func readVersionInfo() {
	var (
		Revision   = "unknown"
		LastCommit time.Time
	)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			Revision = kv.Value
		case "vcs.time":
			LastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		}
	}
	log.Printf("Gobot-%s, %v", Revision, LastCommit)
}

func main() {
	readVersionInfo()

	db, err := storage.Open(os.Getenv("MYSQL_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connection established")

	n, err := db.Migrate()
	if err != nil {
		log.Fatalln(err)
	}
	if n > 0 {
		log.Printf("Applied %d migration(s)", n)
	}

	b, err := bot.NewBot(os.Getenv("BOT_TOKEN"), db)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Logged in as @%s (%d)", b.Me.Username, b.Me.ID)

	if err != nil {
		log.Fatalln(err)
	}

	p, err := bot.NewPlugin(b)
	if err != nil {
		log.Fatalln(err)
	}

	plugins := []bot.IPlugin{
		&plugin.AboutPlugin{Plugin: p},
		&plugin.AllowPlugin{Plugin: p},
		&plugin.CredsPlugin{Plugin: p},
		&plugin.DcryptPlugin{Plugin: p},
		&plugin.EchoPlugin{Plugin: p},
		&plugin.GetFilePlugin{Plugin: p},
		&plugin.IdPlugin{Plugin: p},
		&plugin.ManagerPlugin{Plugin: p},
		&plugin.StatsPlugin{Plugin: p},
	}

	for i, plg := range plugins {
		log.Printf("Registering plugin (%d/%d): %s", i+1, len(plugins), plg.GetName())
		b.RegisterPlugin(plg)
	}

	_, shouldPrintMsgs := os.LookupEnv("PRINT_MSGS")
	if shouldPrintMsgs {
		b.Use(bot.PrintMessage)
	}

	b.Handle(telebot.OnText, b.OnText)
	b.Handle(telebot.OnMedia, b.OnText)
	b.Handle(telebot.OnContact, b.OnText)
	b.Handle(telebot.OnLocation, b.OnText)
	b.Handle(telebot.OnVenue, b.OnText)
	b.Handle(telebot.OnGame, b.OnText)
	b.Handle(telebot.OnDice, b.OnText)
	b.Handle(telebot.OnUserJoined, b.OnUserJoined)
	b.Handle(telebot.OnUserLeft, b.OnUserLeft)
	b.Handle(telebot.OnCallback, b.OnCallback)
	b.Handle(telebot.OnQuery, b.OnInlineQuery)

	b.Handle(telebot.OnEdited, b.NullRoute)
	b.Handle(telebot.OnPinned, b.NullRoute)
	b.Handle(telebot.OnNewGroupTitle, b.NullRoute)
	b.Handle(telebot.OnNewGroupPhoto, b.NullRoute)
	b.Handle(telebot.OnGroupPhotoDeleted, b.NullRoute)
	b.Handle(telebot.OnGroupCreated, b.NullRoute)
	// TODO: Handle edits for getFile replacement file? ein common pluginStruct?

	b.Start()
}
