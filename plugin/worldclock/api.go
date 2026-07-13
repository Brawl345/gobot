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
	sign := "+"
	offset := r.GmtOffset
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}
