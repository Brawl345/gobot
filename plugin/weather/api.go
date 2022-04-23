package weather

import (
	"fmt"
	"strings"
)

type (
	Temperature      float64
	PrecipitationSum float64

	Response struct {
		CurrentWeather struct {
			Temperature Temperature `json:"temperature"`
			Weathercode Weathercode `json:"weathercode"`
		} `json:"current_weather"`
		Daily struct {
			PrecipitationHours []int              `json:"precipitation_hours"`
			PrecipitationSum   []PrecipitationSum `json:"precipitation_sum"`
			Sunrise            []string           `json:"sunrise"`
			Sunset             []string           `json:"sunset"`
			Temperature2MMax   []Temperature      `json:"temperature_2m_max"`
			Temperature2MMin   []Temperature      `json:"temperature_2m_min"`
			Time               []string           `json:"time"`
			Weathercode        []Weathercode      `json:"weathercode"`
		} `json:"daily"`
		Hourly struct {
			Precipitation []float64     `json:"precipitation"`
			Temperature2M []Temperature `json:"temperature_2m"`
			Time          []string      `json:"time"`
			Weathercode   []Weathercode `json:"weathercode"`
		} `json:"hourly"`
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
