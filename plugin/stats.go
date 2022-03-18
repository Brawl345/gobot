package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
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

func (plg *StatsPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
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

	for _, user := range users {
		// TODO: Prozentangabe
		// TODO: Nur Leute in der Gruppe + andere aussortieren
		totalCount += user.MsgCount
		sb.WriteString(
			fmt.Sprintf("<b>%s:</b> %s\n",
				user.FirstName,
				utils.CommaFormat(user.MsgCount),
			),
		)
	}

	sb.WriteString("==============\n")
	sb.WriteString(fmt.Sprintf("<b>TOTAL:</b> %s", utils.CommaFormat(totalCount)))

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
