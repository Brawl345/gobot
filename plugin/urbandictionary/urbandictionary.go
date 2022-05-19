package urbandictionary

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("urbandictionary")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "urbandictionary"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "ud",
			Description: "<Begriff> - Im Urban Dictionary suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/ud(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onUrbanDictionary,
		},
	}
}

func onUrbanDictionary(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	query := c.Matches[1]

	var response Response
	err := utils.GetRequest(fmt.Sprintf(Url, url.QueryEscape(query)), &response)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Failed to search urban dictionary")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if len(response.List) == 0 {
		return c.Reply("❌ Nichts gefunden.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	term := response.List[0]
	sb.WriteString(
		fmt.Sprintf(
			"<b><a href=\"%s\">%s</a></b>\n",
			term.Permalink,
			utils.Escape(term.Word),
		),
	)
	definition := strings.NewReplacer(
		"[", "",
		"]", "",
	).Replace(term.Definition)

	sb.WriteString(utils.Escape(definition))

	if len(term.Example) > 0 {
		example := strings.NewReplacer(
			"[", "",
			"]", "",
		).Replace(term.Example)
		sb.WriteString(fmt.Sprintf("\n\n<i>Beispiel:</i>\n%s", utils.Escape(example)))
	}

	timezone := utils.GermanTimezone()
	sb.WriteString(
		fmt.Sprintf(
			"\n\n<i>Vom %s</i>",
			term.WrittenOn.In(timezone).Format("02.01.2006, 15:04 Uhr"),
		),
	)

	if term.Upvotes > 0 {
		sb.WriteString(fmt.Sprintf(" - 👍 %s", utils.FormatThousand(term.Upvotes)))
	}
	if term.Downvotes > 0 {
		sb.WriteString(fmt.Sprintf(" - 👎 %s", utils.FormatThousand(term.Downvotes)))
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
