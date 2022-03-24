package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"html"
	"log"
	"regexp"
	"strings"
)

type StatsPlugin struct {
	*bot.Plugin
}

func (*StatsPlugin) GetName() string {
	return "stats"
}

func (plg *StatsPlugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/stats(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler:   plg.OnStats,
			GroupOnly: true,
		},
	}
}

func (plg *StatsPlugin) OnStats(c bot.NextbotContext) error {
	users, err := plg.Bot.DB.ChatsUsers.GetAllUsersWithMsgCount(c.Chat())
	if err != nil {
		log.Println(err)
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
					utils.CommaFormat(user.MsgCount),
					percentage,
				),
			)
		}
	}

	sb.WriteString("==============\n")
	if otherMsgs > 0 {
		percentage := (float64(otherMsgs) / float64(totalCount)) * 100
		sb.WriteString(fmt.Sprintf("<b>Andere Nutzer:</b> %s <code>(%.2f %%)</code>\n",
			utils.CommaFormat(otherMsgs),
			percentage),
		)
	}
	sb.WriteString(fmt.Sprintf("<b>GESAMT:</b> %s", utils.CommaFormat(totalCount)))

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
