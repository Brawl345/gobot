package worldclock

import (
	"fmt"
)

type Response struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	ZoneName     string `json:"zoneName"`
	Abbreviation string `json:"abbreviation"`
	GmtOffset    int    `json:"gmtOffset"`
	Timestamp    int64  `json:"timestamp"`
}

func (r *Response) GmtOffsetFormatted() string {
	var sign string
	if r.GmtOffset > 0 {
		sign = "+"
	}
	hours := r.GmtOffset / 3600
	minutes := (r.GmtOffset % 3600) / 60
	formattedStr := fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
	return formattedStr
}
