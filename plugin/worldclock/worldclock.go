package worldclock

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

type Plugin struct {
	credentialService model.CredentialService // https://www.bingmapsportal.com/
	geocodingService  model.GeocodingService
}

var log = logger.New("worldclock")

func New(credentialService model.CredentialService, geocodingService model.GeocodingService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
		geocodingService:  geocodingService,
	}
}

func (p *Plugin) Name() string {
	return "worldclock"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "time",
			Description: "[Ort] - Aktuelle Uhrzeit an diesem Ort",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func (p *Plugin) onTime(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)

	apiKey := p.credentialService.GetKey("bing_maps_api_key")
	if apiKey == "" {
		log.Warn().Msg("bing_maps_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>bing_maps_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	var location string
	if len(c.Matches) > 1 {
		location = c.Matches[1]
	} else {
		location = "Berlin, Deutschland"
	}
	venue, err := p.geocodingService.Geocode(location)
	if err != nil {
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("location", c.Matches[1]).
			Msg("Failed to get coordinates for location")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Koordinaten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "dev.virtualearth.net",
		Path:   fmt.Sprintf("/REST/v1/TimeZone/%f,%f", venue.Location.Latitude, venue.Location.Longitude),
	}

	q := requestUrl.Query()
	q.Set("key", apiKey)
	q.Set("culture", "de-de")

	requestUrl.RawQuery = q.Encode()

	var response Response
	var httpError *httpUtils.HttpError
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Response: &response,
	})
	if err != nil {
		if errors.As(err, &httpError) && httpError.StatusCode == http.StatusNotFound {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", requestUrl.String()).
			Msg("error requesting API")

		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if len(response.ResourceSets) == 0 || len(response.ResourceSets[0].Resources) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	timezone := response.ResourceSets[0].Resources[0].TimeZone

	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b>\n",
			utils.Escape(timezone.IanaTimeZoneId),
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
			utils.Escape(timezone.ConvertedTime.TimeZoneDisplayName),
			utils.Escape(timezone.ConvertedTime.TimeZoneDisplayAbbr),
			utils.Escape(timezone.ConvertedTime.UtcOffsetWithDstFormatted()),
		),
	)

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}
