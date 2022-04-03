package utils

import (
	"golang.org/x/exp/constraints"
	"math"
	"strconv"
	"strings"
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
