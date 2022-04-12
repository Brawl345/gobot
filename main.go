package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	_ "github.com/joho/godotenv/autoload"
)

var log = logger.New("main")

func main() {
	versionInfo, err := utils.ReadVersionInfo()
	if err != nil {
		log.Err(err).Send()
	} else {
		log.Info().Msgf("Gobot-%s, %v", versionInfo.Revision, versionInfo.LastCommit)
	}

	b, err := bot.New()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("Logged in as @%s (%d)", b.Telebot.Me.Username, b.Telebot.Me.ID)

	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	signal.Notify(channel, os.Interrupt, syscall.SIGKILL)
	signal.Notify(channel, os.Interrupt, syscall.SIGINT)
	go func() {
		<-channel
		log.Info().Msg("Stopping...")
		//b.Telebot.Stop()
		os.Exit(0)
	}()

	b.Telebot.Start()
}
