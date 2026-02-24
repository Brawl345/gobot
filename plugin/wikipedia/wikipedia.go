package wikipedia

import (
	"errors"
	"fmt"
	"net"
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

var log = logger.New("wikipedia")

const (
	maxNumDisambiguationList = 5
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "wikipedia"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "wiki",
			Description: "<Begriff> - In der Wikipedia nachschlagen",
		},
		{
			Command:     "wiki_en",
			Description: "<Begriff> - In der englischen Wikipedia nachschlagen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/wiki(?:@%s)? (?P<query>.+)$`, botInfo.Username)),
			HandlerFunc: onArticle,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/wiki_(?P<lang>\w+)(?:@%s)? (?P<query>.+)$`, botInfo.Username)),
			HandlerFunc: onArticle,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)https?://(?P<lang>\w+).(?:m.)?wikipedia.org/wiki/(?P<query>[^\s#]+)(?:#(?P<section>\S+))?`),
			HandlerFunc: onArticle,
		},
	}
}

func onArticle(b *gotgbot.Bot, c plugin.GobotContext) error {
	query := c.NamedMatches["query"]
	lang := c.NamedMatches["lang"]
	if lang == "" {
		lang = "de"
	}
	section := c.NamedMatches["section"]

	query = strings.NewReplacer(
		"_", " ",
		"?", "\\?",
	).Replace(query)
	query = regexWprov.ReplaceAllString(query, "")
	query, err := url.PathUnescape(query)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Failed to unescape query")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s.wikipedia.org", lang),
		Path:   "/w/api.php",
	}

	q := requestUrl.Query()
	q.Set("action", "query")
	q.Set("titles", query)
	q.Set("format", "json")
	q.Set("prop", "info|pageprops|extracts")
	q.Set("redirects", "1")
	q.Set("formatversion", "2")
	q.Set("inprop", "url")
	q.Set("ppprop", "disambiguation")
	if section == "" {
		q.Set("exintro", "1")
	}
	q.Set("explaintext", "1")

	requestUrl.RawQuery = q.Encode()

	var response Response
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Headers:  map[string]string{"User-Agent": "Gobot/1.0 (Telegram Bot; +https://github.com/Brawl345/gobot)"},
		Response: &response,
	})
	if err != nil {
		var noSuchHostErr *net.DNSError
		if errors.As(err, &noSuchHostErr) {
			_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Diese Wikipedia-Sprachversion existiert nicht.", nil)
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Failed to get Wikipedia response")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if len(response.Query.Pages) == 0 {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Artikel nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	article := response.Query.Pages[0]
	if article.Missing {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Artikel nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	if article.Invalid {
		log.Error().
			Str("query", query).
			Str("invalid_reason", article.InvalidReason).
			Msg("Invalid article")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Artikel nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b><a href=\"%s\">%s</a></b>\n",
			article.URL,
			utils.Escape(article.Title),
		),
	)

	if article.Pageprops.Disambiguation {
		// Need to parse the disambiguation page manually
		requestUrl := url.URL{
			Scheme: "https",
			Host:   fmt.Sprintf("%s.wikipedia.org", lang),
			Path:   "/w/api.php",
		}

		q := requestUrl.Query()
		q.Set("action", "query")
		q.Set("titles", article.Title)
		q.Set("format", "json")
		q.Set("prop", "info|pageprops|extracts")
		q.Set("redirects", "1")
		q.Set("formatversion", "2")
		q.Set("inprop", "url")
		q.Set("ppprop", "disambiguation")

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
				Msg("Failed to get disambiugation response")
			_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions())
			return err
		}

		matches := regexDisambiguation.FindAllStringSubmatch(response.Query.Pages[0].Text, -1)
		sb.WriteString("<i>Dies ist eine Begriffsklärungsseite.</i>\n")
		if len(matches) > 0 {
			sb.WriteString("\n<b>Meintest du:</b>\n")
			for i, match := range matches {
				if i == maxNumDisambiguationList {
					break
				}
				articleTitle := strings.TrimSpace(match[1])
				articleTitle = regexHTML.ReplaceAllString(articleTitle, "")
				sb.WriteString(fmt.Sprintf("* %s\n", utils.Escape(articleTitle)))
			}
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
							Text: "...weitere?",
							Url:  article.URL,
						},
					},
				},
			},
		})
		return err
	}

	if section != "" {
		section = strings.ReplaceAll(section, "_", " ")
		section, err = url.PathUnescape(section)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("query", query).
				Str("section", section).
				Msg("Failed to unescape section")
			_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions())
			return err
		}
		matches := regexSection.FindAllStringSubmatch(article.Text, -1)
		for _, match := range matches {
			sectionTitle := strings.TrimSpace(match[1])
			if section == sectionTitle {
				sectionText := strings.TrimSpace(match[2])
				if sectionText == "" {
					break
				}
				sb.WriteString(
					fmt.Sprintf(
						"<b>Abschnitt:</b> <i>%s</i>\n",
						utils.Escape(sectionTitle),
					),
				)
				article.Text = sectionText
				break
			}
		}
	}

	summary := strings.TrimSpace(article.Text)
	if len(summary) > 400 {
		summary = summary[:400] + "..."
	}
	sb.WriteString(
		fmt.Sprintf(
			"%s\n",
			utils.Escape(summary),
		),
	)

	_, err = c.EffectiveMessage.ReplyMessage(b, sb.String(), utils.DefaultSendOptions())
	return err
}
