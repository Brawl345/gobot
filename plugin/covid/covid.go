package covid

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/telebot.v3"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var log = logger.NewLogger("covid")

const BaseUrl = "https://disease.sh/v3/covid-19"

type (
	Plugin struct {
		*bot.Plugin
	}

	countryResult struct {
		Message     null.String `json:"message"`
		Updated     null.Int    `json:"updated"`
		Country     null.String `json:"country"`
		CountryInfo struct {
			Flag null.String `json:"flag"`
		} `json:"countryInfo"`
		Cases                  null.Int    `json:"cases"`
		TodayCases             null.Int    `json:"todayCases"`
		Deaths                 null.Int    `json:"deaths"`
		TodayDeaths            null.Int    `json:"todayDeaths"`
		Recovered              null.Int    `json:"recovered"`
		TodayRecovered         null.Int    `json:"todayRecovered"`
		Active                 null.Int    `json:"active"`
		Critical               null.Int    `json:"critical"`
		CasesPerOneMillion     null.Int    `json:"casesPerOneMillion"`
		DeathsPerOneMillion    null.Int    `json:"deathsPerOneMillion"`
		Tests                  null.Int    `json:"tests"`
		TestsPerOneMillion     null.Int    `json:"testsPerOneMillion"`
		Population             null.Int    `json:"population"`
		Continent              null.String `json:"continent"`
		OneCasePerPeople       null.Int    `json:"oneCasePerPeople"`
		OneDeathPerPeople      null.Int    `json:"oneDeathPerPeople"`
		OneTestPerPeople       null.Int    `json:"oneTestPerPeople"`
		ActivePerOneMillion    null.Float  `json:"activePerOneMillion"`
		RecoveredPerOneMillion null.Float  `json:"recoveredPerOneMillion"`
		CriticalPerOneMillion  null.Float  `json:"criticalPerOneMillion"`
	}

	vaccineResult struct {
		Message  null.String `json:"message"`
		Country  null.String `json:"country"`
		Timeline []struct {
			Total           int64  `json:"total"`
			Daily           int64  `json:"daily"`
			TotalPerHundred int64  `json:"totalPerHundred"`
			DailyPerMillion int64  `json:"dailyPerMillion"`
			Date            string `json:"date"`
		} `json:"timeline"`
	}
)

func (result *countryResult) UpdatedParsed() time.Time {
	return time.Unix(result.Updated.Int64/1000, 0)
}

func (*Plugin) GetName() string {
	return "covid"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/covid(?:@%s)?[ _]([A-z ]+)(?:@%s)?$`,
				plg.Bot.Me.Username,
				plg.Bot.Me.Username),
			),
			Handler: plg.OnCountry,
		},
	}
}

func getVaccineData(country string) (vaccineResult, error) {
	resp, err := http.Get(
		fmt.Sprintf(
			"%s/vaccine/coverage/countries/%s?lastdays=1&fullData=true",
			BaseUrl, url.PathEscape(country),
		),
	)

	if err != nil {
		return vaccineResult{}, err
	}

	if resp.StatusCode == 404 {
		return vaccineResult{}, errors.New("country not found or has no cases")
	}

	if resp.StatusCode != 200 {
		return vaccineResult{}, errors.New("unexpected status code")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return vaccineResult{}, err
	}

	var result vaccineResult
	if err := json.Unmarshal(body, &result); err != nil {
		return vaccineResult{}, err
	}

	if result.Message.Valid {
		return vaccineResult{}, errors.New(result.Message.String)
	}

	return result, nil
}

func (plg *Plugin) OnCountry(c bot.NextbotContext) error {
	c.Notify(telebot.Typing)

	resp, err := http.Get(
		fmt.Sprintf(
			"%s/countries/%s?strict=false&allowNull=true",
			BaseUrl, url.PathEscape(c.Matches[1]),
		),
	)
	if err != nil {
		log.Err(err).Send()
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	if resp.StatusCode == 404 {
		return c.Reply("❌ Das gesuchte Land existiert nicht oder hat keine COVID-Fälle gemeldet.\n"+
			"Bitte darauf achten das Land in <b>Englisch</b> anzugeben!",
			utils.DefaultSendOptions,
		)
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status_code", resp.StatusCode).Msg("Unexpected status code")
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read response body")
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	var result countryResult
	if err := json.Unmarshal(body, &result); err != nil {
		log.Err(err).Msg("Failed to unmarshal response body")
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	if result.Message.Valid {
		log.Error().Str("message", result.Message.String).Msg("Error message found in data")
		return c.Reply(fmt.Sprintf("❌ %s", result.Message.String), utils.DefaultSendOptions)
	}

	var sb strings.Builder
	if result.CountryInfo.Flag.Valid {
		sb.WriteString(utils.EmbedImage(result.CountryInfo.Flag.String))
	}

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Fälle in %s</b>:\n",
			html.EscapeString(result.Country.String),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Mio.)\n",
			utils.CommaFormat(result.Cases.Int64),
			utils.CommaFormat(result.TodayCases.Int64),
			utils.CommaFormat(result.CasesPerOneMillion.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Aktiv:</b> %s\n",
			utils.CommaFormat(result.Active.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Kritisch:</b> %s\n",
			utils.CommaFormat(result.Critical.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s (+ %s)\n",
			utils.CommaFormat(result.Recovered.Int64),
			utils.CommaFormat(result.TodayRecovered.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s) (%s pro Mio.)\n",
			utils.CommaFormat(result.Deaths.Int64),
			utils.CommaFormat(result.TodayDeaths.Int64),
			utils.CommaFormat(result.DeathsPerOneMillion.Int64),
		),
	)

	c.Notify(telebot.Typing)
	vaccine, err := getVaccineData(result.Country.String)
	if err != nil {
		log.Err(err).Msg("Error while getting vaccine data")
	} else {
		sb.WriteString(
			fmt.Sprintf(
				"<b>Impfungen:</b> %s",
				utils.CommaFormat(vaccine.Timeline[0].Total),
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

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply: true,
		ParseMode:         telebot.ModeHTML,
	})

}
