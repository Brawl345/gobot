package weather

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("weather")

type Plugin struct {
	geocodingService models.GeocodingService
	homeService      models.HomeService
}

func New(geocodingService models.GeocodingService, homeService models.HomeService) *Plugin {
	return &Plugin{
		geocodingService: geocodingService,
		homeService:      homeService,
	}
}

func (p *Plugin) Name() string {
	return "weather"
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/w(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onWeather,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/w(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onWeather,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/f(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onForecast,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/f(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onForecast,
		},
	}
}

func (p *Plugin) onWeather(c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)

	var err error
	var venue telebot.Venue
	if len(c.Matches) > 1 {
		venue, err = p.geocodingService.Geocode(c.Matches[1])
	} else {
		venue, err = p.homeService.GetHome(c.Sender())
	}

	if err != nil {
		if errors.Is(err, models.ErrHomeAddressNotSet) {
			return c.Reply("🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions)
		}
		if errors.Is(err, models.ErrAddressNotFound) {
			return c.Reply("❌ Ort nicht gefunden.", utils.DefaultSendOptions)
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error getting location")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	requestUrl := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=weathercode,temperature_2m_max,temperature_2m_min,sunrise,sunset,precipitation_sum,precipitation_hours&hourly=precipitation&current_weather=true&timezone=Europe/Berlin", venue.Location.Lat, venue.Location.Lng)

	var response Response
	err = utils.GetRequest(requestUrl, &response)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting weather")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🌡 <b>Wetter in %s:</b>\n",
			html.EscapeString(venue.Address),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Jetzt:</b> %s %s | %s %s\n",
			response.CurrentWeather.Temperature.String(),
			response.CurrentWeather.Temperature.Icon(),
			response.CurrentWeather.Weathercode.Description(),
			response.CurrentWeather.Weathercode.Icon(),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Heute:</b> Max. %s | Min. %s | %s %s\n",
			response.Daily.Temperature2MMax[0].String(),
			response.Daily.Temperature2MMin[0].String(),
			response.Daily.Weathercode[0].Description(),
			response.Daily.Weathercode[0].Icon(),
		),
	)

	if response.Daily.PrecipitationHours[0] > 0 {
		var plural string
		if response.Daily.PrecipitationHours[0] > 1 {
			plural = "n"
		}

		sb.WriteString(
			fmt.Sprintf(
				"💧 %d Regenstunde%s mit %s Niederschlag\n",
				response.Daily.PrecipitationHours[0],
				plural,
				response.Daily.PrecipitationSum[0].String(),
			),
		)

		var rainyHours []int
		for i, hourlyPrecipitation := range response.Hourly.Precipitation {
			if hourlyPrecipitation > 0 {
				rainyHours = append(rainyHours, i)
			}
			if i >= 23 {
				break
			}
		}

		var rainyHoursCollapsed [][]int
		prevHour := -1
		for i, hour := range rainyHours {
			if i == 0 {
				rainyHoursCollapsed = append(rainyHoursCollapsed, []int{hour})
				prevHour = hour
			} else {
				if hour == prevHour+1 {
					rainyHoursCollapsed[len(rainyHoursCollapsed)-1] = append(rainyHoursCollapsed[len(rainyHoursCollapsed)-1], hour)
				} else {
					rainyHoursCollapsed = append(rainyHoursCollapsed, []int{hour})
				}
				prevHour = hour
			}
		}

		var rainyHoursString strings.Builder
		for i, ints := range rainyHoursCollapsed {
			if len(ints) > 2 {
				rainyHoursString.WriteString(fmt.Sprintf("%d-%d", ints[0], ints[len(ints)-1]))
			} else if len(ints) == 2 {
				rainyHoursString.WriteString(fmt.Sprintf("%d, %d", ints[0], ints[1]))
			} else {
				rainyHoursString.WriteString(fmt.Sprintf("%d", ints[0]))
			}
			if i < len(rainyHoursCollapsed)-1 {
				rainyHoursString.WriteString(", ")
			}
		}

		sb.WriteString(
			fmt.Sprintf(
				"Regen um: %s Uhr\n",
				rainyHoursString.String(),
			),
		)

	}

	sunriseTime, err := time.Parse("2006-01-02T15:04", response.Daily.Sunrise[0])
	if err != nil {
		log.Error().
			Err(err).
			Str("sunriseTime", response.Daily.Sunrise[0]).
			Msg("error parsing sunrise time")
	}
	sunrise := "?"
	if err == nil {
		sunrise = sunriseTime.Format("15:04 Uhr")
	}

	sunsetTime, err := time.Parse("2006-01-02T15:04", response.Daily.Sunset[0])
	if err != nil {
		log.Error().
			Err(err).
			Str("sunsetTime", response.Daily.Sunset[0]).
			Msg("error parsing sunset time")
	}
	sunset := "?"
	if err == nil {
		sunset = sunsetTime.Format("15:04 Uhr")
	}

	sb.WriteString(
		fmt.Sprintf(
			"☀⏫: %s | ☀⏬: %s\n",
			sunrise,
			sunset,
		),
	)

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func (p *Plugin) onForecast(c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)

	var err error
	var venue telebot.Venue
	if len(c.Matches) > 1 {
		venue, err = p.geocodingService.Geocode(c.Matches[1])
	} else {
		venue, err = p.homeService.GetHome(c.Sender())
	}

	if err != nil {
		if errors.Is(err, models.ErrHomeAddressNotSet) {
			return c.Reply("🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions)
		}
		if errors.Is(err, models.ErrAddressNotFound) {
			return c.Reply("❌ Ort nicht gefunden.", utils.DefaultSendOptions)
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error getting location")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	requestUrl := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=weathercode,temperature_2m_max,temperature_2m_min&timezone=Europe/Berlin", venue.Location.Lat, venue.Location.Lng)

	var response Response
	err = utils.GetRequest(requestUrl, &response)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting weather")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🌡 <b>Wettervorhersage für %s:</b>\n",
			html.EscapeString(venue.Address),
		),
	)

	for day := range response.Daily.Time {
		forecast, err := response.Daily.Forecast(day)
		if err != nil {
			guid := xid.New().String()
			log.Error().
				Err(err).
				Str("guid", guid).
				Msg("error constructing forecast")
			sb.WriteString(fmt.Sprintf("❌ Fehler: <code>%s</code>", guid))
		} else {
			sb.WriteString(forecast)
		}
		sb.WriteString("\n")
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
