package covid

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("covid")

const (
	BaseUrl       = "https://disease.sh/v3/covid-19"
	MyCountry     = "Germany" // Country that will definitely be shown in the top list
	TopListPlaces = 10
)

type (
	Plugin struct{}

	Result struct {
		Message                null.String `json:"message"`
		Updated                null.Int    `json:"updated"`
		Cases                  null.Int    `json:"cases"`
		TodayCases             null.Int    `json:"todayCases"`
		Deaths                 null.Int    `json:"deaths"`
		TodayDeaths            null.Int    `json:"todayDeaths"`
		Recovered              null.Int    `json:"recovered"`
		TodayRecovered         null.Int    `json:"todayRecovered"`
		Active                 null.Int    `json:"active"`
		Critical               null.Int    `json:"critical"`
		CasesPerOneMillion     null.Int    `json:"casesPerOneMillion"`
		Tests                  null.Int    `json:"tests"`
		TestsPerOneMillion     null.Float  `json:"testsPerOneMillion"`
		Population             null.Int    `json:"population"`
		OneCasePerPeople       null.Int    `json:"oneCasePerPeople"`
		OneDeathPerPeople      null.Int    `json:"oneDeathPerPeople"`
		OneTestPerPeople       null.Int    `json:"oneTestPerPeople"`
		ActivePerOneMillion    null.Float  `json:"activePerOneMillion"`
		RecoveredPerOneMillion null.Float  `json:"recoveredPerOneMillion"`
		CriticalPerOneMillion  null.Float  `json:"criticalPerOneMillion"`
	}

	allResult struct {
		*Result
		AffectedCountries   int        `json:"affectedCountries"`
		DeathsPerOneMillion null.Float `json:"deathsPerOneMillion"`
	}

	countryResult struct {
		*Result
		Country     null.String `json:"country"`
		CountryInfo struct {
			Flag null.String `json:"flag"`
		} `json:"countryInfo"`
		Continent           null.String `json:"continent"`
		DeathsPerOneMillion null.Int    `json:"deathsPerOneMillion"`
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

func New() *Plugin {
	return &Plugin{}
}

func (countryResult *countryResult) GetRankingText(place int) string {
	return fmt.Sprintf(
		"%d. <b>%s:</b> %s Gesamt (+ %s); %s aktiv\n",
		place,
		countryResult.Country.String,
		utils.FormatThousand(countryResult.Cases.Int64),
		utils.FormatThousand(countryResult.TodayCases.Int64),
		utils.FormatThousand(countryResult.Active.Int64),
	)
}

func (result *Result) UpdatedParsed() time.Time {
	return time.Unix(result.Updated.Int64/1000, 0)
}

func (*Plugin) Name() string {
	return "covid"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/covid(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: OnRun,
		},
		&plugin.CommandHandler{
			Trigger: regexp.MustCompile(fmt.Sprintf(`^/covid(?:@%s)?[ _]([A-z ]+)(?:@%s)?$`,
				botInfo.Username,
				botInfo.Username),
			),
			HandlerFunc: OnCountry,
		},
	}
}

func OnCountry(c plugin.NextbotContext) error {
	c.Notify(telebot.Typing)

	var httpError *utils.HttpError
	var result countryResult

	err := utils.GetRequest(
		fmt.Sprintf(
			"%s/countries/%s?strict=false&allowNull=true",
			BaseUrl, url.PathEscape(c.Matches[1]),
		),
		&result,
	)

	if err != nil {
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 404 {
				return c.Reply("❌ Das gesuchte Land existiert nicht oder hat keine COVID-Fälle gemeldet.\n"+
					"Bitte darauf achten das Land in <b>Englisch</b> anzugeben!",
					utils.DefaultSendOptions,
				)
			} else {
				log.Error().Int("status_code", httpError.StatusCode).Msg("Unexpected status code")
			}
		} else {
			log.Err(err).Send()
		}

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
			utils.FormatThousand(result.Cases.Int64),
			utils.FormatThousand(result.TodayCases.Int64),
			utils.FormatThousand(result.CasesPerOneMillion.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Aktiv:</b> %s\n",
			utils.FormatThousand(result.Active.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Kritisch:</b> %s\n",
			utils.FormatThousand(result.Critical.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s (+ %s)\n",
			utils.FormatThousand(result.Recovered.Int64),
			utils.FormatThousand(result.TodayRecovered.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s) (%s pro Mio.)\n",
			utils.FormatThousand(result.Deaths.Int64),
			utils.FormatThousand(result.TodayDeaths.Int64),
			utils.FormatThousand(result.DeathsPerOneMillion.Int64),
		),
	)

	c.Notify(telebot.Typing)
	var vaccine vaccineResult
	err = utils.GetRequest(
		fmt.Sprintf(
			"%s/vaccine/coverage/countries/%s?lastdays=1&fullData=true",
			BaseUrl, url.PathEscape(result.Country.String),
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

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply: true,
		ParseMode:         telebot.ModeHTML,
	})

}

func OnRun(c plugin.NextbotContext) error {
	c.Notify(telebot.Typing)

	resultCh := make(chan allResult)
	allCountriesCh := make(chan []countryResult)
	hasErrorCh := make(chan bool)

	go func() {
		var countryResults []countryResult
		err := utils.GetRequest(
			fmt.Sprintf(
				"%s/countries?sort=cases&allowNull=true",
				BaseUrl,
			),
			&countryResults,
		)
		if err != nil {
			log.Err(err).Str("on", "countries").Send()
			close(allCountriesCh)
			return
		}
		allCountriesCh <- countryResults
	}()

	go func() {
		var all allResult
		err := utils.GetRequest(
			fmt.Sprintf(
				"%s/all?allowNull=true",
				BaseUrl,
			),
			&all,
		)
		if err != nil {
			log.Err(err).Str("on", "all").Send()
			hasErrorCh <- true
			close(resultCh)
			return
		}
		resultCh <- all
		close(hasErrorCh)
	}()

	result, allCountries, hasError := <-resultCh, <-allCountriesCh, <-hasErrorCh

	if hasError {
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Fälle weltweit (%d Länder):</b>\n",
			result.AffectedCountries,
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Million)\n",
			utils.FormatThousand(result.Cases.Int64),
			utils.FormatThousand(result.TodayCases.Int64),
			utils.FormatThousand(result.CasesPerOneMillion.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Aktiv:</b> %s (%s pro Million)\n",
			utils.FormatThousand(result.Active.Int64),
			utils.RoundAndFormatThousand(result.ActivePerOneMillion.Float64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s\n",
			utils.FormatThousand(result.Recovered.Int64),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s) (%s pro Million)\n\n",
			utils.FormatThousand(result.Deaths.Int64),
			utils.FormatThousand(result.TodayDeaths.Int64),
			utils.RoundAndFormatThousand(result.DeathsPerOneMillion.Float64),
		),
	)

	myCountryIndex := 0

	for i, country := range allCountries {
		if country.Country.String == MyCountry {
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
			result.UpdatedParsed().Format("02.01.2006, 15:04:05"),
		),
	)

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply: true,
		ParseMode:         telebot.ModeHTML,
	})
}
