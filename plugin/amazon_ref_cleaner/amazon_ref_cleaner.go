package amazon_ref_cleaner

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.New("amazon_ref_cleaner")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "amazon_ref_cleaner"
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:amzn\.to/[A-Za-z\d]+|amazon\.[\w.]+/\S+)`),
			HandlerFunc: onAmazonLink,
		},
	}
}

func onAmazonLink(c plugin.GobotContext) error {
	var links []string
	for _, entity := range utils.AnyEntities(c.Message()) {
		if entity.Type == telebot.EntityURL {
			amazonUrl, err := url.Parse(c.Message().EntityText(entity))

			if err != nil {
				log.Err(err).
					Str("url", c.Message().EntityText(entity)).
					Msg("Failed to parse amazon url")
				continue
			}

			if amazonUrl.Hostname() == "amzn.to" {
				req, err := http.NewRequest("GET", amazonUrl.String(), nil) // HEAD requests lead to 405 :(
				req.Header.Set("User-Agent", utils.UserAgent)               // Amazon blocks unknown user agents

				if err != nil {
					log.Err(err).
						Str("url", amazonUrl.String()).
						Msg("Failed to create request")
					continue
				}

				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				resp, err := client.Do(req)

				if err != nil {
					log.Err(err).
						Str("url", amazonUrl.String()).
						Msg("Failed to send request")
					continue
				}

				if resp.StatusCode != 301 {
					log.Error().
						Int("status_code", resp.StatusCode).
						Msg("Got non-301 status code")
					continue
				}

				amazonUrl, err = resp.Location()
				if err != nil {
					log.Error().
						Interface("headers", resp.Header).
						Str("url", amazonUrl.String()).
						Msg("Failed to parse location header")
					continue
				}
			}

			if amazonUrl.Query().Has("tag") || amazonUrl.Query().Has("linkId") {
				amazonUrl.RawQuery = ""
				links = append(links, amazonUrl.String())
			}
		}
	}

	if len(links) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>Ohne Ref:</b>\n")
	for _, link := range links {
		sb.WriteString(link + "\n")
	}

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableNotification:   true,
		DisableWebPagePreview: true,
		ParseMode:             telebot.ModeHTML,
	})
}
