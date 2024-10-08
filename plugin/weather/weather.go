package weather

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("weather")

type Plugin struct {
	geocodingService model.GeocodingService
	homeService      model.HomeService
}

func New(geocodingService model.GeocodingService, homeService model.HomeService) *Plugin {
	return &Plugin{
		geocodingService: geocodingService,
		homeService:      homeService,
	}
}

func (p *Plugin) Name() string {
	return "weather"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "f",
			Description: "[Ort] - Wettervorhersage",
		},
		{
			Command:     "fh",
			Description: "[Ort] - 24-Stunden-Wettervorhersage",
		},
		{
			Command:     "w",
			Description: "[Ort] - Aktuelles Wetter",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/fh(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onHourlyForecast,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/fh(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onHourlyForecast,
		},
	}
}

func (p *Plugin) onWeather(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionFindLocation, nil)

	var err error
	var venue gotgbot.Venue
	if len(c.Matches) > 1 {
		venue, err = p.geocodingService.Geocode(c.Matches[1])
	} else {
		venue, err = p.homeService.GetHome(c.EffectiveUser)
	}

	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			_, err := c.EffectiveMessage.Reply(b, "🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions())
			return err
		}
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
			return err
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error getting location")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	requestUrl := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=weathercode,temperature_2m_max,temperature_2m_min,sunrise,sunset,precipitation_sum,precipitation_hours&hourly=precipitation&current_weather=true&timezone=Europe/Berlin", venue.Location.Latitude, venue.Location.Longitude)

	var response Response
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl,
		Response: &response,
	})
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting weather")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🌡 <b>Wetter in %s:</b>\n",
			utils.Escape(venue.Address),
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

	if response.Daily.PrecipitationHours[0] > 0.0 {
		var plural string
		if response.Daily.PrecipitationHours[0] > 1.0 {
			plural = "n"
		}

		precipitationHours := fmt.Sprintf("%.2f", response.Daily.PrecipitationHours[0])
		precipitationHours = strings.NewReplacer(".00", "", ".", ",").Replace(precipitationHours)

		sb.WriteString(
			fmt.Sprintf(
				"💧 %s Regenstunde%s mit %s Niederschlag\n",
				precipitationHours,
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

		rainyHoursCollapsed := make([][]int, 0)
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
			"☀⏫ %s | ☀⏬ %s\n",
			sunrise,
			sunset,
		),
	)

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onForecast(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionFindLocation, nil)

	var err error
	var venue gotgbot.Venue
	if len(c.Matches) > 1 {
		venue, err = p.geocodingService.Geocode(c.Matches[1])
	} else {
		venue, err = p.homeService.GetHome(c.EffectiveUser)
	}

	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			_, err := c.EffectiveMessage.Reply(b, "🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions())
			return err
		}
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
			return err
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error getting location")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	requestUrl := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=weathercode,temperature_2m_max,temperature_2m_min&timezone=Europe/Berlin", venue.Location.Latitude, venue.Location.Longitude)

	var response Response
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl,
		Response: &response,
	})
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting weather")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🌡 <b>Wettervorhersage für %s:</b>\n",
			utils.Escape(venue.Address),
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

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onHourlyForecast(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionFindLocation, nil)

	var err error
	var venue gotgbot.Venue
	if len(c.Matches) > 1 {
		venue, err = p.geocodingService.Geocode(c.Matches[1])
	} else {
		venue, err = p.homeService.GetHome(c.EffectiveUser)
	}

	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			_, err := c.EffectiveMessage.Reply(b, "🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions())
			return err
		}
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Ort nicht gefunden.", utils.DefaultSendOptions())
			return err
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error getting location")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	requestUrl := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,weathercode&timezone=Europe/Berlin", venue.Location.Latitude, venue.Location.Longitude)

	var response Response
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl,
		Response: &response,
	})
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting weather")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🌡 <b>24-Stunden-Vorhersage für %s:</b>\n",
			utils.Escape(venue.Address),
		),
	)

	for hour := range response.Hourly.Time {
		currentHour := time.Now().Hour()

		forecast, err := response.Hourly.Forecast(hour + currentHour)
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

		if hour == 24 {
			break
		}
	}

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}
