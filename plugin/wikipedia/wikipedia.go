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
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "wiki",
			Description: "<Begriff> - In der Wikipedia nachschlagen",
		},
		{
			Text:        "wiki_en",
			Description: "<Begriff> - In der englischen Wikipedia nachschlagen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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

func onArticle(c plugin.GobotContext) error {
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
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
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
	err = utils.GetRequest(requestUrl.String(), &response)
	if err != nil {
		var noSuchHostErr *net.DNSError
		if errors.As(err, &noSuchHostErr) {
			return c.Reply("❌ Diese Wikipedia-Sprachversion existiert nicht.")
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("Failed to get Wikipedia response")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.Query.Pages) == 0 {
		return c.Reply("❌ Artikel nicht gefunden.", utils.DefaultSendOptions)
	}

	article := response.Query.Pages[0]
	if article.Missing {
		return c.Reply("❌ Artikel nicht gefunden.", utils.DefaultSendOptions)
	}

	if article.Invalid {
		log.Error().
			Str("query", query).
			Str("invalid_reason", article.InvalidReason).
			Msg("Invalid article")
		return c.Reply("❌ Artikel nicht gefunden.", utils.DefaultSendOptions)
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
			Host:   "de.wikipedia.org",
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
		err := utils.GetRequest(requestUrl.String(), &response)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("query", query).
				Msg("Failed to get disambiugation response")
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
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

		return c.Reply(sb.String(), &telebot.SendOptions{
			AllowWithoutReply:     true,
			DisableWebPagePreview: true,
			DisableNotification:   true,
			ParseMode:             telebot.ModeHTML,
			ReplyMarkup: &telebot.ReplyMarkup{
				InlineKeyboard: [][]telebot.InlineButton{
					{
						{
							Text: "...weitere?",
							URL:  article.URL,
						},
					},
				},
			},
		})
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
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
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

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
