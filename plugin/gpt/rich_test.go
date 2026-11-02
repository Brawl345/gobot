package gpt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func TestSanitizeRich(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`Klick <tg-button type="callback_data" data="bi:1">hier</tg-button>.`, "Klick hier."},
		{"<tg-button-row align=\"left\">\n<tg-button type=\"copy_text\" text=\"x\"/>\n</tg-button-row>", "\n\n"},
		{"Hallo [Max](tg://user?id=123) und <a href=\"tg://user?id=5\">Moritz</a>", "Hallo Max und Moritz"},
		{"[Link](https://example.com)", "[Link](https://example.com)"},
		{`<tg-time datetime="2026-11-06T20:15:00+01:00" format="wDT">Do., 20:15</tg-time>`, `<tg-time unix="1793992500" format="wDT">Do., 20:15</tg-time>`},
		{`<tg-time format="t" datetime="2026-07-01T12:00">12 Uhr</tg-time>`, `<tg-time format="t" unix="1782900000">12 Uhr</tg-time>`},
		{`<tg-time datetime="morgen">morgen</tg-time>`, "morgen"},
		{`<tg-time unix="1647531900" format="r">x</tg-time>`, `<tg-time unix="1647531900" format="r">x</tg-time>`},
	}
	for _, tt := range tests {
		if got := sanitizeRich(tt.in); got != tt.want {
			t.Errorf("sanitizeRich(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStripRichMedia(t *testing.T) {
	in := "Text\n\n<tg-collage>\n\n![](https://a/b.jpg \"cap\")\n![x](https://a/c.jpg)\n<img src=\"https://a/d.jpg\"/>\n\n</tg-collage>\n\n<tg-slideshow>\n<figure><img src=\"https://a/f.jpg\" tg-spoiler/><figcaption>Cap<cite>a.de</cite></figcaption></figure>\n</tg-slideshow>\n<figure><tg-map lat=\"1\" long=\"2\" zoom=\"3\"/></figure>\nInline ![](https://a/e.jpg) bleibt\nEnde"
	want := "Text\n\n\n\n\n<figure><tg-map lat=\"1\" long=\"2\" zoom=\"3\"/></figure>\nInline ![](https://a/e.jpg) bleibt\nEnde"
	if got := stripRichMedia(in); got != want {
		t.Errorf("stripRichMedia() = %q, want %q", got, want)
	}
}

func TestEmbedRichMedia(t *testing.T) {
	media := map[string]fetchedImage{
		"https://a/b.jpg":     {mimeType: "image/jpeg", data: []byte{1}},
		"https://a/c.jpg":     {mimeType: "image/jpeg", data: []byte{2}},
		"https://a/g.jpg?a&b": {mimeType: "image/jpeg", data: []byte{3}},
	}
	in := "![](https://a/b.jpg \"cap\")\n![x](<https://a/c.jpg>)\n![](https://a/b.jpg)\n![](https://other/d.jpg)\n" +
		`<figure><img src="https://a/g.jpg?a&amp;b" tg-spoiler/><figcaption>C<cite>a.de</cite></figcaption></figure>` + "\n" +
		`<IMG alt="x" SRC='https://a/b.jpg'/>` + "\n" + `<img src="https://other/h.jpg"/>`
	want := "![](tg://photo?id=img1 \"cap\")\n![x](<tg://photo?id=img2>)\n![](tg://photo?id=img1)\n![](https://other/d.jpg)\n" +
		`<figure><img src="tg://photo?id=img3" tg-spoiler/><figcaption>C<cite>a.de</cite></figcaption></figure>` + "\n" +
		`<IMG alt="x" SRC='tg://photo?id=img1'/>` + "\n" + `<img src="https://other/h.jpg"/>`
	got, files := embedRichMedia(in, media)
	if got != want {
		t.Errorf("embedRichMedia() = %q, want %q", got, want)
	}
	if len(files) != 3 || files[0].Id != "img1" || files[1].Id != "img2" || files[2].Id != "img3" {
		t.Errorf("embedRichMedia() files = %+v", files)
	}
}

func TestBuildRichMarkdown(t *testing.T) {
	segs := []citedSegment{
		{text: "Preis **5 €**."},
		{url: "https://example.com/?a=1&b=2", num: 1},
	}
	got := buildRichMarkdown(segs, "⚒️🔎", `(<a href="https://x.org">x.org</a>)`)
	want := "⚒️🔎 Preis **5 €**.<a href=\"https://example.com/?a=1&b=2\">[1]</a>\n\n<footer>(<a href=\"https://x.org\">x.org</a>)</footer>"
	if got != want {
		t.Errorf("buildRichMarkdown() = %q, want %q", got, want)
	}

	for _, body := range []string{"## Titel", "- Punkt", "1. Punkt", "| a |", "```go", "![](x)", "<details>", "> Zitat", "$$x$$"} {
		got := buildRichMarkdown([]citedSegment{{text: body}}, "⚒️", "")
		if want := "⚒️\n\n" + body; got != want {
			t.Errorf("buildRichMarkdown(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestReplyContentRich(t *testing.T) {
	raw := `{"message_id":1,"date":0,"chat":{"id":1,"type":"group"},"rich_message":{"blocks":[` +
		`{"type":"paragraph","text":["⚒️🔎 Preis ",{"type":"bold","text":"5 €"},{"type":"url","text":"[1]","url":"https://example.com/"},"."]},` +
		`{"type":"collage","blocks":[{"type":"photo","photo":[{"file_id":"small","file_unique_id":"s","width":90,"height":90},{"file_id":"big","file_unique_id":"b","width":800,"height":600}],"caption":{"text":"Logo"}}]},` +
		`{"type":"footer","text":["(",{"type":"url","text":"example.com","url":"https://example.com/"},")"]}]}}`
	var m gotgbot.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	text, photos := replyContent(&m)
	t.Log(text)
	if !strings.Contains(text, "https://example.com/") || !strings.Contains(text, "5 €") {
		t.Errorf("replyContent() text = %q", text)
	}
	if len(photos) != 2 || photos[1].FileId != "big" {
		t.Errorf("replyContent() photos = %+v", photos)
	}
}
