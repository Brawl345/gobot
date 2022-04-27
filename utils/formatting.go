package utils

import (
	"math"
	"strconv"
	"strings"

	"github.com/sosodev/duration"
	"golang.org/x/exp/constraints"
)

func RoundAndFormatThousand(n float64) string {
	return FormatThousand(int64(math.Round(n)))
}

func FormatThousand[T constraints.Integer](n T) string {
	// TODO: Replace with https://stackoverflow.com/a/46811454/3146627
	// 	when https://youtrack.jetbrains.com/issue/GO-5841 is fixed
	in := strconv.FormatInt(int64(n), 10)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits--
	}
	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = '.'
		}
	}
}

func EmbedImage(url string) string {
	var sb strings.Builder

	sb.WriteString("<a href=\"")
	sb.WriteString(url)
	sb.WriteString("\">")
	sb.WriteString("\u200c") // ZWNJ
	sb.WriteString("</a>")

	return sb.String()
}

func EmbedGUID(guid string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("(<code>")
	sb.WriteString(guid)
	sb.WriteString("</code>)")
	return sb.String()
}

func FullName(firstName, lastName string) string {
	var sb strings.Builder
	sb.WriteString(firstName)
	if lastName != "" {
		sb.WriteString(" ")
		sb.WriteString(lastName)
	}
	return sb.String()
}

func HumanizeDuration(d *duration.Duration) string {
	var sb strings.Builder

	if d.Years > 0 {
		sb.WriteString(strconv.Itoa(int(d.Years)))
		sb.WriteString("y")
	}

	if d.Months > 0 {
		sb.WriteString(strconv.Itoa(int(d.Months)))
		sb.WriteString("M")
	}

	if d.Weeks > 0 {
		sb.WriteString(strconv.Itoa(int(d.Weeks)))
		sb.WriteString("w")
	}

	if d.Days > 0 {
		sb.WriteString(strconv.Itoa(int(d.Days)))
		sb.WriteString("d")
	}

	if d.Hours > 0 {
		sb.WriteString(strconv.Itoa(int(d.Hours)))
		sb.WriteString("h")
	}

	if d.Minutes > 0 {
		sb.WriteString(strconv.Itoa(int(d.Minutes)))
		sb.WriteString("m")
	}

	if d.Seconds > 0 {
		sb.WriteString(strconv.Itoa(int(d.Seconds)))
		sb.WriteString("s")
	}

	return sb.String()
}

func LocalizeDatestring(date string) string {
	return strings.NewReplacer(
		"January", "Januar",
		"February", "Februar",
		"March", "März",
		"May", "Mai",
		"June", "Juni",
		"July", "Juli",
		"October", "Oktober",
		"December", "Dezember",
		"Mar", "Mär",
		"Oct", "Okt",
		"Dec", "Dez",
		"Monday", "Montag",
		"Tuesday", "Dienstag",
		"Wednesday", "Mittwoch",
		"Thursday", "Donnerstag",
		"Friday", "Freitag",
		"Saturday", "Samstag",
		"Sunday", "Sonntag",
		"Mon", "Mo",
		"Tue", "Di",
		"Wed", "Mi",
		"Thu", "Do",
		"Fri", "Fr",
		"Sat", "Sa",
		"Sun", "So",
	).Replace(date)
}
