package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Brawl345/gobot/utils"
	_ "github.com/joho/godotenv/autoload"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model/sql"
)

var log = logger.New("main")

func main() {
	versionInfo, err := utils.ReadVersionInfo()
	if err != nil {
		log.Err(err).Send()
	} else {
		log.Info().Msgf("Gobot-%s, %v", versionInfo.Revision, versionInfo.LastCommit)
	}

	db, err := sql.New()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	b, err := bot.New(db)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("Logged in as @%s (%d)", b.GoTgBot.Username, b.GoTgBot.Id)

	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-channel
		log.Info().Msg("Stopping...")
		os.Exit(0)
	}()

	b.Start()
}
