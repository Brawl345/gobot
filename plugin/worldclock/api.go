package worldclock

import (
	"strings"
	"time"
)

type (
	Response struct {
		ResourceSets []struct {
			EstimatedTotal int `json:"estimatedTotal"`
			Resources      []struct {
				TimeZone struct {
					Abbreviation   string        `json:"abbreviation"`
					IanaTimeZoneId string        `json:"ianaTimeZoneId"`
					ConvertedTime  ConvertedTime `json:"convertedTime"`
				} `json:"timeZone"`
			} `json:"resources"`
		} `json:"resourceSets"`
		StatusCode        int    `json:"statusCode"`
		StatusDescription string `json:"statusDescription"`
	}

	ConvertedTime struct {
		LocalTime           string `json:"localTime"`
		UtcOffsetWithDst    string `json:"utcOffsetWithDst"`
		TimeZoneDisplayName string `json:"timeZoneDisplayName"`
		TimeZoneDisplayAbbr string `json:"timeZoneDisplayAbbr"`
	}
)

func (c *ConvertedTime) ParsedTime() (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05", c.LocalTime)
}

func (c *ConvertedTime) UtcOffsetWithDstFormatted() string {
	if !strings.HasPrefix(c.UtcOffsetWithDst, "-") {
		return "+" + c.UtcOffsetWithDst
	}
	return c.UtcOffsetWithDst
}
