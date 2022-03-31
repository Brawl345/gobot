package utils

import (
	"strconv"
	"strings"
)

// CommaFormat https://stackoverflow.com/a/31046325/3146627
func CommaFormat(n int64) string {
	in := strconv.FormatInt(n, 10)
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
