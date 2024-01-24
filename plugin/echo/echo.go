package echo

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "echo"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "echo",
			Description: "<Text> - Echo... echo... echo...",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/e(?:cho)?(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onEcho,
		},
	}
}

func onEcho(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := c.EffectiveMessage.Reply(b, c.Matches[1], &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
	})
	return err
}
