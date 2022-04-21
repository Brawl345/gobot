package covid

import (
	"fmt"
	"time"

	"github.com/Brawl345/gobot/utils"
	"gopkg.in/guregu/null.v4"
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
