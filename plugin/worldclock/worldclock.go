package worldclock

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"gopkg.in/telebot.v3"
)

type Plugin struct {
	apiKey string // https://www.bingmapsportal.com/
}

func New(credentialService models.CredentialService) *Plugin {
	apiKey, err := credentialService.GetKey("bing_maps_api_key")
	if err != nil {
		log.Warn().Msg("bing_maps_api_key not found")
	}

	return &Plugin{
		apiKey: apiKey,
	}
}

func (p *Plugin) Name() string {
	return "worldclock"
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/time?(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onTime,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/time?(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onTime,
		},
	}
}

func (p *Plugin) onTime(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "dev.virtualearth.net",
		Path:   "/REST/v1/TimeZone/",
	}

	q := requestUrl.Query()
	q.Set("key", p.apiKey)

	var location string
	if len(c.Matches) > 1 {
		location = c.Matches[1]
	} else {
		location = "Berlin"
	}
	q.Set("query", location)
	q.Set("culture", "de-de")

	requestUrl.RawQuery = q.Encode()

	var response Response
	var httpError *utils.HttpError
	err := utils.GetRequest(requestUrl.String(), &response)
	if err != nil {
		if errors.As(err, &httpError) && httpError.StatusCode == 404 {
			return c.Reply("❌ Ort nicht gefunden.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", requestUrl.String()).
			Msg("error requesting API")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.ResourceSets) == 0 ||
		len(response.ResourceSets[0].Resources) == 0 ||
		len(response.ResourceSets[0].Resources[0].TimeZoneAtLocation) == 0 ||
		len(response.ResourceSets[0].Resources[0].TimeZoneAtLocation[0].TimeZone) == 0 {
		return c.Reply("❌ Ort nicht gefunden.", utils.DefaultSendOptions)
	}

	var sb strings.Builder
	timezone := response.ResourceSets[0].Resources[0].TimeZoneAtLocation[0].TimeZone[0]

	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b>\n",
			html.EscapeString(timezone.IanaTimeZoneId),
		),
	)

	parsedTime, err := timezone.ConvertedTime.ParsedTime()
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("localTime", timezone.ConvertedTime.LocalTime).
			Msg("error converting local time from API")
		sb.WriteString(fmt.Sprintf("❌ Fehler bei der Konvertierung.%s\n", utils.EmbedGUID(guid)))
	} else {
		sb.WriteString(
			fmt.Sprintf(
				"🕒 %s\n",
				utils.LocalizeDatestring(parsedTime.Format("Monday, 02. January 2006, 15:04:05 Uhr")),
			),
		)
	}

	sb.WriteString(
		fmt.Sprintf(
			"<i>%s (%s, UTC%s)</i>",
			html.EscapeString(timezone.ConvertedTime.TimeZoneDisplayName),
			html.EscapeString(timezone.ConvertedTime.TimeZoneDisplayAbbr),
			html.EscapeString(timezone.ConvertedTime.UtcOffsetWithDstFormatted()),
		),
	)

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
