package bot

import "strings"

func EmbedImage(url string) string {
	var sb strings.Builder

	sb.WriteString("<a href=\"")
	sb.WriteString(url)
	sb.WriteString("\">")
	sb.WriteString("\u200c") // ZWNJ
	sb.WriteString("</a>")

	return sb.String()
}
