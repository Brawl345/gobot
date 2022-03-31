package covid

import (
	"encoding/json"
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/telebot.v3"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var log = logger.NewLogger("covid")

const BaseUrl = "https://disease.sh/v3/covid-19"

type (
	Plugin struct {
		*bot.Plugin
	}

	countryResult struct {
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
)

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

func (plg *Plugin) OnCountry(c bot.NextbotContext) error {
	c.Notify(telebot.Typing)

	resp, err := http.Get(
		fmt.Sprintf(
			"%s/countries/%s?strict=false&allowNull=true",
			BaseUrl, c.Matches[1],
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

	var sb strings.Builder
	if result.CountryInfo.Flag.Valid {
		sb.WriteString(bot.EmbedImage(result.CountryInfo.Flag.String))
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

	// TODO: Vaccines (mit goroutine und text editieren...?)
	// TODO: Updated datum

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply: true,
		ParseMode:         telebot.ModeHTML,
	})

}
