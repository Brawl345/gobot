package main

import (
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin/about"
	"github.com/Brawl345/gobot/plugin/allow"
	"github.com/Brawl345/gobot/plugin/covid"
	"github.com/Brawl345/gobot/plugin/creds"
	"github.com/Brawl345/gobot/plugin/dcrypt"
	"github.com/Brawl345/gobot/plugin/echo"
	"github.com/Brawl345/gobot/plugin/getfile"
	"github.com/Brawl345/gobot/plugin/id"
	"github.com/Brawl345/gobot/plugin/manager"
	"github.com/Brawl345/gobot/plugin/stats"
	"github.com/Brawl345/gobot/storage"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/telebot.v3"
	"os"
	"runtime/debug"
	"time"
)

var log = logger.NewLogger("main")

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
	log.Info().Msgf("Gobot-%s, %v", Revision, LastCommit)
}

func main() {
	readVersionInfo()

	db, err := storage.Open(os.Getenv("MYSQL_URL"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msg("Database connection established")

	n, err := db.Migrate()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if n > 0 {
		log.Info().Msgf("Applied %d migration(s)", n)
	}

	b, err := bot.NewBot(os.Getenv("BOT_TOKEN"), db)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Str("component", "core").Msgf("Logged in as @%s (%d)", b.Me.Username, b.Me.ID)

	if err != nil {
		log.Fatal().Err(err).Send()
	}

	p, err := bot.NewPlugin(b)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	plugins := []bot.IPlugin{
		&about.Plugin{Plugin: p},
		&allow.Plugin{Plugin: p},
		&covid.Plugin{Plugin: p},
		&creds.Plugin{Plugin: p},
		&dcrypt.Plugin{Plugin: p},
		&echo.Plugin{Plugin: p},
		&getfile.Plugin{Plugin: p},
		&id.Plugin{Plugin: p},
		&manager.Plugin{Plugin: p},
		&stats.Plugin{Plugin: p},
	}

	for i, plg := range plugins {
		log.Info().Msgf("Registering plugin (%d/%d): %s", i+1, len(plugins), plg.GetName())
		b.RegisterPlugin(plg)
	}

	_, shouldPrintMsgs := os.LookupEnv("PRINT_MSGS")
	if shouldPrintMsgs {
		b.Use(bot.PrintMessage)
	}

	b.Handle(telebot.OnText, b.OnText)
	b.Handle(telebot.OnEdited, b.OnText)
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

	b.Handle(telebot.OnPinned, b.NullRoute)
	b.Handle(telebot.OnNewGroupTitle, b.NullRoute)
	b.Handle(telebot.OnNewGroupPhoto, b.NullRoute)
	b.Handle(telebot.OnGroupPhotoDeleted, b.NullRoute)
	b.Handle(telebot.OnGroupCreated, b.NullRoute)

	b.OnError = bot.OnError

	b.Start()
}
