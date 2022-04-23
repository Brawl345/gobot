package gps

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("gps")

type (
	Plugin struct {
		geocodingService models.GeocodingService
	}
	Response struct {
		DisplayName string `json:"display_name"`
	}
)

func New(geocodingService models.GeocodingService) *Plugin {
	return &Plugin{
		geocodingService: geocodingService,
	}
}

func (p *Plugin) Name() string {
	return "gps"
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/(?:gps|map)(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onGPS,
		},
		&plugin.CommandHandler{
			Trigger:     telebot.OnLocation,
			HandlerFunc: p.onLocation,
		},
		&plugin.CommandHandler{
			Trigger:     telebot.OnVenue,
			HandlerFunc: p.onLocation,
		},
	}
}

func (p *Plugin) onGPS(c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)
	venue, err := p.geocodingService.Geocode(c.Matches[1])
	if err != nil {
		if errors.Is(err, models.ErrAddressNotFound) {
			return c.Reply("❌ Ort nicht gefunden.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("location", c.Matches[1]).
			Msg("Failed to get coordinates for location")
		return c.Reply(fmt.Sprintf("❌ Fehler beim Abrufen der Koordinaten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	return c.Reply(&venue, utils.DefaultSendOptions)
}

func (p *Plugin) onLocation(c plugin.GobotContext) error {
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

	if c.Message().Location != nil {
		lat = strconv.FormatFloat(float64(c.Message().Location.Lat), 'f', -1, 32)
		lon = strconv.FormatFloat(float64(c.Message().Location.Lng), 'f', -1, 32)
	} else {
		lat = strconv.FormatFloat(float64(c.Message().Venue.Location.Lat), 'f', -1, 32)
		lon = strconv.FormatFloat(float64(c.Message().Venue.Location.Lng), 'f', -1, 32)
	}

	q.Set("lat", lat)
	q.Set("lon", lon)

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := utils.GetRequestWithHeader(
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
		return c.Reply(fmt.Sprintf(
			"<a href=\"https://maps.google.com/maps?q=%s,%s&ll=%s,%s&z=16\">%s</a>",
			lat, lon, lat, lon, html.EscapeString(response.DisplayName),
		), utils.DefaultSendOptions)
	}
	return nil
}
