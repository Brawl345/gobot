package alive

import (
	"fmt"
	"math/rand"
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "alive"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)^Bot\??$`),
			HandlerFunc: onAliveCheck,
		},
	}
}

func onAliveCheck(b *gotgbot.Bot, c plugin.GobotContext) error {
	emojis := []string{
		"🤗",
		"💩",
		"👌",
		"👍",
		"🖕",
		"🍌",
		"😐",
		"🤨",
		"\U0001FAE1",
	}

	randomEmoji := emojis[rand.Intn(len(emojis))]

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, randomEmoji, &tgUtils.ReactionFallbackOpts{
		Fallback: fmt.Sprintf("<b>Ich bin da, %s!</b>", utils.Escape(c.EffectiveSender.FirstName())),
		SendMessageOpts: &gotgbot.SendMessageOpts{
			ParseMode: gotgbot.ParseModeHTML,
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
			},
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled: true,
			},
		},
	})
}
