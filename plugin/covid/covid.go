package covid

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
	"golang.org/x/sync/errgroup"
)

const (
	BaseUrl       = "https://disease.sh/v3/covid-19"
	MyCountry     = "Germany" // Country that will definitely be shown in the top list
	TopListPlaces = 10
)

var log = logger.New("covid")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "covid"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "covid",
			Description: "[Ort] - COVID-19-Statistik",
		},
		{
			Command:     "covid_germany",
			Description: "COVID-19-Statistik für Deutschland",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/covid(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: OnRun,
		},
		&plugin.CommandHandler{
			Trigger: regexp.MustCompile(fmt.Sprintf(`(?i)^/covid(?:@%s)?[ _]([A-z ]+)(?:@%s)?$`,
				botInfo.Username,
				botInfo.Username),
			),
			HandlerFunc: OnCountry,
		},
	}
}

func OnCountry(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

	var httpError *httpUtils.HttpError
	var result countryResult

	err := httpUtils.GetRequest(
		fmt.Sprintf(
			"%s/countries/%s?strict=false",
			BaseUrl, url.PathEscape(c.Matches[1]),
		),
		&result,
	)

	if err != nil {
		guid := xid.New().String()
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 404 {
				_, err := c.EffectiveMessage.Reply(
					b,
					fmt.Sprintf("❌ Das gesuchte Land existiert nicht oder hat keine COVID-Fälle gemeldet.\n"+
						"Bitte darauf achten das Land in <b>Englisch</b> anzugeben!%s", utils.EmbedGUID(guid)),
					utils.DefaultSendOptions,
				)
				return err
			} else {
				log.Error().
					Str("guid", guid).
					Int("status_code", httpError.StatusCode).
					Msg("Unexpected status code")
			}
		} else {
			log.Err(err).
				Str("guid", guid).
				Send()
		}

		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Bei der Anfrage ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
		return err
	}

	if result.Message != "" {
		log.Error().Str("message", result.Message).Msg("Error message found in data")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ %s", result.Message), utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder
	if result.CountryInfo.Flag != "" {
		sb.WriteString(utils.EmbedImage(result.CountryInfo.Flag))
	}

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Fälle in %s</b>:\n",
			utils.Escape(result.Country),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Mio.)\n",
			utils.FormatThousand(result.Cases),
			utils.FormatThousand(result.TodayCases),
			utils.FormatThousand(result.CasesPerOneMillion),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Aktiv:</b> %s\n",
			utils.FormatThousand(result.Active),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Kritisch:</b> %s\n",
			utils.FormatThousand(result.Critical),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s (+ %s)\n",
			utils.FormatThousand(result.Recovered),
			utils.FormatThousand(result.TodayRecovered),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s) (%s pro Mio.)\n",
			utils.FormatThousand(result.Deaths),
			utils.FormatThousand(result.TodayDeaths),
			utils.FormatThousand(result.DeathsPerOneMillion),
		),
	)

	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)
	var vaccine vaccineResult
	err = httpUtils.GetRequest(
		fmt.Sprintf(
			"%s/vaccine/coverage/countries/%s?lastdays=1&fullData=true",
			BaseUrl, url.PathEscape(result.Country),
		),
		&vaccine,
	)

	if err != nil {
		log.Err(err).Msg("Error while getting vaccine data")
	} else if len(vaccine.Timeline) > 0 {
		sb.WriteString(
			fmt.Sprintf(
				"<b>Impfungen:</b> %s",
				utils.FormatThousand(vaccine.Timeline[0].Total),
			),
		)
		vaccineDate, err := time.Parse("1/2/06", vaccine.Timeline[0].Date)
		if err != nil {
			log.Err(err).Msg("Failed to parse date")
		} else {
			sb.WriteString(fmt.Sprintf(" (vom %s)", vaccineDate.Format("02.01.2006")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(
		fmt.Sprintf(
			"\n<i>Zuletzt aktualisiert: %s Uhr</i>",
			result.UpdatedParsed().Format("02.01.2006, 15:04:05"),
		),
	)

	_, err = c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		ParseMode:       gotgbot.ParseModeHTML,
	})
	return err

}

func OnRun(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, _ := errgroup.WithContext(ctx)

	var allCountries []countryResult
	eg.Go(func() error {
		return httpUtils.GetRequest(
			fmt.Sprintf(
				"%s/countries?sort=cases",
				BaseUrl,
			),
			&allCountries,
		)
	})

	var all allResult
	err := httpUtils.GetRequest(
		fmt.Sprintf(
			"%s/all",
			BaseUrl,
		),
		&all,
	)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("on", "all").
			Msg("Failed to get 'all' data")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Daten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Fälle weltweit (%d Länder):</b>\n",
			all.AffectedCountries,
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Million)\n",
			utils.FormatThousand(all.Cases),
			utils.FormatThousand(all.TodayCases),
			utils.FormatThousand(all.CasesPerOneMillion),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Aktiv:</b> %s (%s pro Million)\n",
			utils.FormatThousand(all.Active),
			utils.RoundAndFormatThousand(all.ActivePerOneMillion),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s\n",
			utils.FormatThousand(all.Recovered),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s) (%s pro Million)\n\n",
			utils.FormatThousand(all.Deaths),
			utils.FormatThousand(all.TodayDeaths),
			utils.RoundAndFormatThousand(all.DeathsPerOneMillion),
		),
	)

	err = eg.Wait()
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("on", "countries").
			Msg("Failed to get 'all countries' data")
		sb.WriteString(fmt.Sprintf("❌ Fehler beim Abrufen aller Länder.%s", utils.EmbedGUID(guid)))

		_, err := c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
			ParseMode:       gotgbot.ParseModeHTML,
		})
		return err
	}

	myCountryIndex := 0

	for i, country := range allCountries {
		if country.Country == MyCountry {
			myCountryIndex = i
			break
		}
	}

	if myCountryIndex < TopListPlaces { // My country is in the top list
		for i := 0; i < TopListPlaces; i++ {
			country := allCountries[i]
			sb.WriteString(country.GetRankingText(i + 1))
		}
	} else { // My country is not in the toplist - post the whole toplist and append my country + the one after that
		for i := 0; i < TopListPlaces; i++ {
			country := allCountries[i]
			sb.WriteString(country.GetRankingText(i + 1))
		}

		if myCountryIndex != TopListPlaces {
			sb.WriteString("...\n")
		}

		sb.WriteString(allCountries[myCountryIndex].GetRankingText(myCountryIndex + 1))
		sb.WriteString(allCountries[myCountryIndex+1].GetRankingText(myCountryIndex + 2))
	}

	sb.WriteString(
		fmt.Sprintf(
			"\n<i>Zuletzt aktualisiert: %s Uhr</i>",
			all.UpdatedParsed().Format("02.01.2006, 15:04:05"),
		),
	)

	_, err = c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		ParseMode:       gotgbot.ParseModeHTML,
	})
	return err
}
