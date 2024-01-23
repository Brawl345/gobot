package delmsg

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
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

	if c.EffectiveMessage.ReplyToMessage.From == nil || c.EffectiveMessage.ReplyToMessage.From.Id != b.Id {
		log.Debug().Msg("Message is not a reply to bot")
		return nil
	}

	_, err := b.DeleteMessages(c.EffectiveChat.Id, []int64{
		c.EffectiveMessage.ReplyToMessage.MessageId,
		c.EffectiveMessage.MessageId,
	}, nil)

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete the messages")
	}

	return nil
}
