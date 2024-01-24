package covid

import (
	"fmt"
	"time"

	"github.com/Brawl345/gobot/utils"
)

type (
	Result struct {
		Message                string  `json:"message"`
		Updated                int64   `json:"updated"`
		Cases                  int     `json:"cases"`
		TodayCases             int     `json:"todayCases"`
		Deaths                 int     `json:"deaths"`
		TodayDeaths            int     `json:"todayDeaths"`
		Recovered              int     `json:"recovered"`
		TodayRecovered         int     `json:"todayRecovered"`
		Active                 int     `json:"active"`
		Critical               int     `json:"critical"`
		CasesPerOneMillion     int     `json:"casesPerOneMillion"`
		Tests                  int     `json:"tests"`
		TestsPerOneMillion     float64 `json:"testsPerOneMillion"`
		Population             int     `json:"population"`
		OneCasePerPeople       int     `json:"oneCasePerPeople"`
		OneDeathPerPeople      int     `json:"oneDeathPerPeople"`
		OneTestPerPeople       int     `json:"oneTestPerPeople"`
		ActivePerOneMillion    float64 `json:"activePerOneMillion"`
		RecoveredPerOneMillion float64 `json:"recoveredPerOneMillion"`
		CriticalPerOneMillion  float64 `json:"criticalPerOneMillion"`
	}

	allResult struct {
		*Result
		AffectedCountries   int     `json:"affectedCountries"`
		DeathsPerOneMillion float64 `json:"deathsPerOneMillion"`
	}

	countryResult struct {
		*Result
		Country     string `json:"country"`
		CountryInfo struct {
			Flag string `json:"flag"`
		} `json:"countryInfo"`
		Continent           string `json:"continent"`
		DeathsPerOneMillion int    `json:"deathsPerOneMillion"`
	}

	vaccineResult struct {
		Message  string `json:"message"`
		Country  string `json:"country"`
		Timeline []struct {
			Total           int64  `json:"total"`
			Daily           int64  `json:"daily"`
			TotalPerHundred int64  `json:"totalPerHundred"`
			DailyPerMillion int64  `json:"dailyPerMillion"`
			Date            string `json:"date"`
		} `json:"timeline"`
	}
)

func (countryResult *countryResult) GetRankingText(place int) string {
	return fmt.Sprintf(
		"%d. <b>%s:</b> %s Gesamt (+ %s); %s aktiv\n",
		place,
		countryResult.Country,
		utils.FormatThousand(countryResult.Cases),
		utils.FormatThousand(countryResult.TodayCases),
		utils.FormatThousand(countryResult.Active),
	)
}

func (result *Result) UpdatedParsed() time.Time {
	return utils.TimestampToTime(result.Updated / 1000)
}
