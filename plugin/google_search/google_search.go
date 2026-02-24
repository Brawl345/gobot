package google_search

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("google_search")

type Plugin struct {
	credentialService model.CredentialService
}

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (p *Plugin) Name() string {
	return "google_search"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "g",
			Description: "<Suchbegriff> - Auf Google suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/g(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onGoogleSearch,
		},
	}
}

func (p *Plugin) onGoogleSearch(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	apiKey := p.credentialService.GetKey("google_api_key")
	if apiKey == "" {
		log.Warn().Msg("google_api_key not found")
		_, err := c.EffectiveMessage.ReplyMessage(b,
			"❌ <code>google_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	searchEngineID := p.credentialService.GetKey("google_search_engine_id")
	if searchEngineID == "" {
		log.Warn().Msg("google_search_engine_id not found")
		_, err := c.EffectiveMessage.ReplyMessage(b,
			"❌ <code>google_search_engine_id</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	query := c.Matches[1]

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "customsearch.googleapis.com",
		Path:   "/customsearch/v1",
	}

	q := requestUrl.Query()
	q.Set("key", apiKey)
	q.Set("cx", searchEngineID)
	q.Set("q", query)
	q.Set("hl", "de")
	q.Set("gl", "de")
	q.Set("num", "7")
	q.Set("safe", "active")
	q.Set("fields", "queries/request/searchTerms,searchInformation/formattedTotalResults,items(title, link, displayLink)")

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Response: &response,
	})

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Error while requesting google search")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Items) == 0 {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Es wurden keine Ergebnisse gefunden.", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	for _, item := range response.Items {
		sb.WriteString(
			fmt.Sprintf(
				"<a href=\"%s\">%s</a> - <code>%s</code>\n",
				item.Link,
				utils.Escape(item.Title),
				utils.Escape(item.DisplayLink),
			),
		)
	}

	_, err = c.EffectiveMessage.ReplyMessage(b, sb.String(), &gotgbot.SendMessageOpts{
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text: fmt.Sprintf("%s Ergebnisse", strings.ReplaceAll(response.SearchInformation.FormattedTotalResults, ",", ".")),
						Url:  fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(query)),
					},
				},
			},
		},
	})

	return err
}
