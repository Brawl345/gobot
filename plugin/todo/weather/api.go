package weather

import (
	"fmt"
	"strings"
	"time"

	"github.com/Brawl345/gobot/utils"
)

type (
	Temperature      float64
	PrecipitationSum float64

	Response struct {
		CurrentWeather struct {
			Temperature Temperature `json:"temperature"`
			Weathercode Weathercode `json:"weathercode"`
		} `json:"current_weather"`
		Daily  Daily  `json:"daily"`
		Hourly Hourly `json:"hourly"`
	}

	Daily struct {
		PrecipitationHours []float32          `json:"precipitation_hours"`
		PrecipitationSum   []PrecipitationSum `json:"precipitation_sum"`
		Sunrise            []string           `json:"sunrise"`
		Sunset             []string           `json:"sunset"`
		Temperature2MMax   []Temperature      `json:"temperature_2m_max"`
		Temperature2MMin   []Temperature      `json:"temperature_2m_min"`
		Time               []string           `json:"time"`
		Weathercode        []Weathercode      `json:"weathercode"`
	}

	Hourly struct {
		Precipitation []float64     `json:"precipitation"`
		Temperature2M []Temperature `json:"temperature_2m"`
		Time          []string      `json:"time"`
		Weathercode   []Weathercode `json:"weathercode"`
	}
)

func (precipitationSum PrecipitationSum) String() string {
	temp := fmt.Sprintf("%.2f mm", precipitationSum)
	return strings.ReplaceAll(temp, ".", ",")
}

func (temperature Temperature) String() string {
	temp := fmt.Sprintf("%.1f°C", temperature)
	return strings.ReplaceAll(temp, ".", ",")
}

func (temperature Temperature) Icon() string {
	if temperature <= 10 {
		return "\U0001F976" // 🥶
	} else if temperature <= 20 {
		return "🙂"
	} else if temperature < 30 {
		return "🤩"
	} else if temperature < 40 {
		return "\U0001F975" // 🥵
	} else {
		return "🤬"
	}
}

func (daily *Daily) Forecast(day int) (string, error) {
	if day > len(daily.Time) {
		return "", fmt.Errorf("day %d is out of range", day)
	}

	var sb strings.Builder

	if day == 0 {
		sb.WriteString("<b>Heute:</b> ")
	} else if day == 1 {
		sb.WriteString("<b>Morgen:</b> ")
	} else {
		dateParsed, err := time.Parse("2006-01-02", daily.Time[day])
		if err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("<b>%s:</b> ", utils.LocalizeDatestring(dateParsed.Format("Mon, 2.01"))))
	}

	sb.WriteString(
		fmt.Sprintf(
			"☀ %s",
			daily.Temperature2MMax[day].String(),
		),
	)
	sb.WriteString(" | ")
	sb.WriteString(
		fmt.Sprintf(
			"🌙 %s",
			daily.Temperature2MMin[day].String(),
		),
	)
	sb.WriteString(" | ")
	sb.WriteString(
		fmt.Sprintf(
			"%s %s",
			daily.Weathercode[day].Icon(),
			daily.Weathercode[day].Description(),
		),
	)

	return sb.String(), nil
}

func (hourly *Hourly) Forecast(hour int) (string, error) {
	if hour > len(hourly.Time) {
		return "", fmt.Errorf("hour %d is out of range", hour)
	}

	var sb strings.Builder

	parsedHour, err := time.Parse("2006-01-02T15:04", hourly.Time[hour])
	if err != nil {
		return "", err
	}
	sb.WriteString(
		fmt.Sprintf(
			"<b>%s Uhr</b>",
			parsedHour.Format("15:04"),
		),
	)
	sb.WriteString(" | ")

	sb.WriteString(hourly.Temperature2M[hour].String())

	sb.WriteString(" | ")
	sb.WriteString(
		fmt.Sprintf(
			"%s %s",
			hourly.Weathercode[hour].Icon(),
			hourly.Weathercode[hour].Description(),
		),
	)

	return sb.String(), nil
}
