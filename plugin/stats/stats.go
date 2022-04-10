package stats

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("stats")

type Plugin struct {
	chatsUsersService models.ChatsUsersService
}

func New(chatsUsersService models.ChatsUsersService) *Plugin {
	return &Plugin{
		chatsUsersService: chatsUsersService,
	}
}

func (*Plugin) Name() string {
	return "stats"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/stats(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnStats,
			GroupOnly:   true,
		},
	}
}

func (plg *Plugin) OnStats(c plugin.GobotContext) error {
	users, err := plg.chatsUsersService.GetAllUsersWithMsgCount(c.Chat())
	if err != nil {
		log.Err(err).Int64("chat_id", c.Chat().ID).Msg("Failed to get statistics")
		return c.Reply("❌ Fehler beim Abrufen der Statistiken.", utils.DefaultSendOptions)
	}

	if len(users) == 0 {
		return c.Reply("<i>Es wurden noch keine Statistiken erstellt.</i>", utils.DefaultSendOptions)
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
		if user.InGroup && user.MsgCount > 0 {
			sb.WriteString(
				fmt.Sprintf("<b>%s:</b> %s <code>(%.2f %%)</code>\n",
					html.EscapeString(user.GetFullName()),
					utils.FormatThousand(user.MsgCount),
					percentage,
				),
			)
		}
	}

	sb.WriteString("==============\n")
	if otherMsgs > 0 {
		percentage := (float64(otherMsgs) / float64(totalCount)) * 100
		sb.WriteString(fmt.Sprintf("<b>Andere Nutzer:</b> %s <code>(%.2f %%)</code>\n",
			utils.FormatThousand(otherMsgs),
			percentage),
		)
	}
	sb.WriteString(fmt.Sprintf("<b>GESAMT:</b> %s", utils.FormatThousand(totalCount)))

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
