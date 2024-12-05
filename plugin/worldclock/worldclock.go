package worldclock

import (
	"errors"
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

type Plugin struct {
	credentialService model.CredentialService
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
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	apiKey := p.credentialService.GetKey("timezonedb_api_key")
	if apiKey == "" {
		log.Warn().Msg("timezonedb_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>timezonedb_api_key</code> fehlt.",
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
		Host:   "api.timezonedb.com",
		Path:   "/v2.1/get-time-zone",
	}

	q := requestUrl.Query()
	q.Set("key", apiKey)
	q.Set("format", "json")
	q.Set("by", "position")
	q.Set("fields", "zoneName,abbreviation,gmtOffset,timestamp")
	q.Set("lat", fmt.Sprintf("%f", venue.Location.Latitude))
	q.Set("lng", fmt.Sprintf("%f", venue.Location.Longitude))

	requestUrl.RawQuery = q.Encode()

	var response Response
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Response: &response,
	})
	if err != nil {
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

	if response.Status != "OK" {
		log.Error().
			Str("status", response.Status).
			Str("message", response.Message).
			Msg("got unexpected response from API")
		_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b> <i>(%s, UTC%s)</i>\n",
			utils.Escape(response.ZoneName),
			utils.Escape(response.Abbreviation),
			utils.Escape(response.GmtOffsetFormatted()),
		),
	)

	parsedTime := utils.TimestampToTime(response.Timestamp).UTC()
	sb.WriteString(
		fmt.Sprintf(
			"🕒 %s",
			utils.LocalizeDatestring(parsedTime.Format("Monday, 02. January 2006, 15:04:05 Uhr")),
		),
	)

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}
