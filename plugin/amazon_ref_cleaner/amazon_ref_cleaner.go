package amazon_ref_cleaner

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var log = logger.New("amazon_ref_cleaner")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "amazon_ref_cleaner"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:amzn\.to/[A-Za-z\d]+|amazon\.[\w.]+/\S+)`),
			HandlerFunc: onAmazonLink,
		},
	}
}

func onAmazonLink(b *gotgbot.Bot, c plugin.GobotContext) error {
	var links []string
	for _, entity := range utils.AnyEntities(c.EffectiveMessage) {
		if utils.EntityType(entity.Type) == utils.EntityTypeURL {
			amazonUrl, err := url.Parse(c.EffectiveMessage.ParseEntity(entity).Url)

			if err != nil {
				log.Err(err).
					Str("url", c.EffectiveMessage.ParseEntity(entity).Url).
					Msg("Failed to parse amazon url")
				continue
			}

			if amazonUrl.Hostname() == "amzn.to" {
				req, err := http.NewRequest("GET", amazonUrl.String(), nil) // HEAD requests lead to 405 :(

				if err != nil {
					log.Err(err).
						Str("url", amazonUrl.String()).
						Msg("Failed to create request")
					continue
				}

				req.Header.Set("User-Agent", utils.UserAgent) // Amazon blocks unknown user agents
				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
					Timeout: 10 * time.Second,
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

				fullLink, err := resp.Location()
				if err != nil {
					log.Error().
						Interface("headers", resp.Header).
						Str("url", amazonUrl.String()).
						Msg("Failed to parse location header")
					continue
				}
				amazonUrl = fullLink
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

	_, err := c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
	})
	return err
}
