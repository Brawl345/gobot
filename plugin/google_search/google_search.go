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
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("google_search")

type Plugin struct {
	apiKey         string
	searchEngineID string
}

func New(credentialService model.CredentialService) *Plugin {
	apiKey, err := credentialService.GetKey("google_api_key")
	if err != nil {
		log.Warn().Msg("google_api_key not found")
	}

	searchEngineID, err := credentialService.GetKey("google_search_engine_id")
	if err != nil {
		log.Warn().Msg("google_search_engine_id not found")
	}

	return &Plugin{
		apiKey:         apiKey,
		searchEngineID: searchEngineID,
	}
}

func (p *Plugin) Name() string {
	return "google_search"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "g",
			Description: "<Suchbegriff> - Auf Google suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/g(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onGoogleSearch,
		},
	}
}

func (p *Plugin) onGoogleSearch(c plugin.GobotContext) error {
	query := c.Matches[1]

	_ = c.Notify(telebot.Typing)
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "customsearch.googleapis.com",
		Path:   "/customsearch/v1",
	}

	q := requestUrl.Query()
	q.Set("key", p.apiKey)
	q.Set("cx", p.searchEngineID)
	q.Set("q", query)
	q.Set("hl", "de")
	q.Set("gl", "de")
	q.Set("num", "7")
	q.Set("safe", "active")
	q.Set("fields", "queries/request/searchTerms,searchInformation/formattedTotalResults,items(title, link, displayLink)")

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := utils.GetRequest(requestUrl.String(), &response)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Error while requesting google search")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if len(response.Items) == 0 {
		return c.Reply("❌ Es wurden keine Ergebnisse gefunden.", utils.DefaultSendOptions)
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

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
		ReplyMarkup: &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{
					{
						Text: fmt.Sprintf("%s Ergebnisse", strings.ReplaceAll(response.SearchInformation.FormattedTotalResults, ",", ".")),
						URL:  fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(query)),
					},
				},
			},
		},
	})
}
