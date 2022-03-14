package plugin

import (
	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
	"regexp"
)

func Register() *bot.Plugin {
	return &bot.Plugin{
		Name: "about",
		Handlers: []bot.Handler{
			{
				Command: regexp.MustCompile(`^/about$`),
				Handler: OnAbout,
			},
		},
	}
}

func OnAbout(b *bot.Nextbot, c telebot.Context) error {
	return c.Reply("About plugin")
}
