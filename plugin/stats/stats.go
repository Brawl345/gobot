package stats

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("stats")

type Plugin struct {
	chatsUsersService model.ChatsUsersService
}

func New(chatsUsersService model.ChatsUsersService) *Plugin {
	return &Plugin{
		chatsUsersService: chatsUsersService,
	}
}

func (*Plugin) Name() string {
	return "stats"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "stats",
			Description: "Chat-Statistiken anzeigen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/stats(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnStats,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) OnStats(b *gotgbot.Bot, c plugin.GobotContext) error {
	users, err := p.chatsUsersService.GetAllUsersWithMsgCount(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to get statistics")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Statistiken.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if len(users) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "<i>Es wurden noch keine Statistiken erstellt.</i>", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	totalCount := int64(0)
	otherMsgs := int64(0)

	for _, user := range users {
		totalCount += user.MsgCount
		if !user.InGroup {
			otherMsgs += user.MsgCount
		}
	}

	for _, user := range users {
		percentage := (float64(user.MsgCount) / float64(totalCount)) * 100
		percentageString := fmt.Sprintf("%.2f", percentage)
		percentageString = strings.ReplaceAll(percentageString, ".", ",")
		if user.InGroup && user.MsgCount > 0 {
			sb.WriteString(
				fmt.Sprintf("<b>%s:</b> %s <code>(%s %%)</code>\n",
					utils.Escape(user.GetFullName()),
					utils.FormatThousand(user.MsgCount),
					percentageString,
				),
			)
		}
	}

	sb.WriteString("==============\n")
	if otherMsgs > 0 {
		percentage := (float64(otherMsgs) / float64(totalCount)) * 100
		percentageString := fmt.Sprintf("%.2f", percentage)
		percentageString = strings.ReplaceAll(percentageString, ".", ",")
		sb.WriteString(fmt.Sprintf("<b>Andere Nutzer:</b> %s <code>(%s %%)</code>\n",
			utils.FormatThousand(otherMsgs),
			percentageString),
		)
	}
	sb.WriteString(fmt.Sprintf("<b>GESAMT:</b> %s", utils.FormatThousand(totalCount)))

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}
