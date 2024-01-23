package delmsg

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
)

var log = logger.New("delmsg")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "delmsg"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/del(?:ete)?(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: deleteMsg,
			AdminOnly:   true,
			GroupOnly:   true,
		},
	}
}

func deleteMsg(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !utils.IsReply(c.EffectiveMessage) {
		log.Debug().Msg("Message is not a reply")
		return nil
	}

	if c.EffectiveMessage.ReplyTo.Sender == nil || c.EffectiveMessage.ReplyTo.Sender.ID != c.Bot().Me.ID {
		log.Debug().Msg("Message is not a reply to bot")
		return nil
	}

	err := c.Bot().Delete(c.EffectiveMessage.ReplyTo)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete message")
	}

	err = c.Delete()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to delete command, probably older than 48 hours or no privileges")
	}

	return nil
}
