package alive

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "alive"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)^Bot\??$`),
			HandlerFunc: onAliveCheck,
		},
	}
}

func onAliveCheck(c plugin.GobotContext) error {
	return c.Reply(
		fmt.Sprintf("<b>Ich bin da, %s!</b>", utils.Escape(c.Sender().FirstName)),
		utils.DefaultSendOptions,
	)
}
