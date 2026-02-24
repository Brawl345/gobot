package urbandictionary

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("urbandictionary")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "urbandictionary"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "ud",
			Description: "<Begriff> - Im Urban Dictionary suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/ud(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onUrbanDictionary,
		},
	}
}

func onUrbanDictionary(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)
	query := c.Matches[1]

	var response Response
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      fmt.Sprintf(Url, url.QueryEscape(query)),
		Response: &response,
	})
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Failed to search urban dictionary")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.List) == 0 {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Nichts gefunden.", utils.DefaultSendOptions())
		return err
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

	_, err = c.EffectiveMessage.ReplyMessage(b, sb.String(), utils.DefaultSendOptions())
	return err
}
