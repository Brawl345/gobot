package main

import (
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/storage"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/telebot.v3"
	"log"
	"os"
)

func main() {
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

	plugins := []bot.IPlugin{
		&plugin.AboutPlugin{},
	}

	for _, plg := range plugins {
		b.RegisterPlugin(plg)
	}

	b.Handle(telebot.OnText, b.OnText)

	//b.Bot.Use(h.PrettyPrint())

	//b.Bot.Handle(telebot.OnText, h.OnText)

	b.Start()
}
