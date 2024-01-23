package gps

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("gps")

type (
	Plugin struct {
		geocodingService model.GeocodingService
	}
	Response struct {
		DisplayName string `json:"display_name"`
	}
)

func New(geocodingService model.GeocodingService) *Plugin {
	return &Plugin{
		geocodingService: geocodingService,
	}
}

func (p *Plugin) Name() string {
	return "gps"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "map",
			Description: "<Ort> - Ort auf der Karte anzeigen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/(?:gps|map)(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onGPS,
		},
		&plugin.CommandHandler{
			Trigger:     utils.LocationMsg,
			HandlerFunc: p.onLocation,
		},
		&plugin.CommandHandler{
			Trigger:     utils.VenueMsg,
			HandlerFunc: p.onLocation,
		},
	}
}

func (p *Plugin) onGPS(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionFindLocation, nil)
	venue, err := p.geocodingService.Geocode(c.Matches[1])
	if err != nil {
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions)
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("location", c.Matches[1]).
			Msg("Failed to get coordinates for location")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Koordinaten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	_, err = b.SendVenue(c.EffectiveChat.Id, venue.Location.Latitude, venue.Location.Longitude, venue.Title, venue.Address, &gotgbot.SendVenueOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		DisableNotification: true,
	})
	return err
}

func (p *Plugin) onLocation(b *gotgbot.Bot, c plugin.GobotContext) error {
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "nominatim.openstreetmap.org",
		Path:   "/reverse.php",
	}

	q := requestUrl.Query()
	q.Set("zoom", "16")
	q.Set("accept-language", "de")
	q.Set("limit", "1")
	q.Set("format", "jsonv2")

	var lat string
	var lon string

	if c.EffectiveMessage.Location != nil {
		lat = strconv.FormatFloat(c.EffectiveMessage.Location.Latitude, 'f', -1, 32)
		lon = strconv.FormatFloat(c.EffectiveMessage.Location.Longitude, 'f', -1, 32)
	} else {
		lat = strconv.FormatFloat(c.EffectiveMessage.Venue.Location.Latitude, 'f', -1, 32)
		lon = strconv.FormatFloat(c.EffectiveMessage.Venue.Location.Longitude, 'f', -1, 32)
	}

	q.Set("lat", lat)
	q.Set("lon", lon)

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"User-Agent": "Gobot for Telegram",
		},
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", requestUrl.String()).
			Msg("Failed to reverse geocode location")
		return nil
	}

	if response.DisplayName != "" {
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf(
			"<a href=\"https://maps.google.com/maps?q=%s,%s&ll=%s,%s&z=16\">%s</a>",
			lat, lon, lat, lon, utils.Escape(response.DisplayName),
		), utils.DefaultSendOptions)
		return err
	}
	return nil
}
