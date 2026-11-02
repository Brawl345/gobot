package gpt

import (
	"bytes"
	"fmt"
	"html"
	"mime/multipart"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

const MaxRichMessageLength = 32768

var (
	richImageURL   = regexp.MustCompile(`(!\[[^\]\n]*\]\(\s*<?)([^\s)>]+)`)
	richImgSrc     = regexp.MustCompile(`(?i)(<img\b[^>]*?\bsrc\s*=\s*["'])([^"']+)`)
	richMediaLine  = regexp.MustCompile(`(?im)^[ \t]*(?:!\[[^\]\n]*\]\([^)\n]*\)|<img\b[^>\n]*>(?:</img>)?|<figure>[ \t]*<img\b[^\n]*</figure>|</?tg-(?:collage|slideshow)>)[ \t]*$\n?`)
	richBlockStart = regexp.MustCompile("^(?:#{1,6}\\s|[-*+]\\s|\\d+[.)]\\s|>|\\||```|!\\[|<|\\$\\$)")

	richButton       = regexp.MustCompile(`(?is)<tg-button\b[^>]*?(?:/>|>(.*?)</tg-button\s*>)`)
	richButtonRow    = regexp.MustCompile(`(?i)</?tg-button-row\b[^>]*>`)
	richMarkdownUser = regexp.MustCompile(`(?i)\[([^\]\n]*)\]\(\s*<?tg://user\?[^)\n]*\)`)
	richHTMLUser     = regexp.MustCompile(`(?is)<a\s[^>]*href\s*=\s*["']?tg://user\?[^>]*>(.*?)</a\s*>`)
	richTimeDatetime = regexp.MustCompile(`(?is)<tg-time\s([^>]*?)datetime\s*=\s*"([^"]*)"([^>]*)>(.*?)</tg-time\s*>`)
	datetimeLayouts  = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"}
	berlinLocation   = sync.OnceValue(func() *time.Location {
		loc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			return time.Local
		}
		return loc
	})

	RichInstruction = `## Output format

Your answer is rendered as Telegram Rich Markdown. Most answers are plain prose without any formatting, which looks like a normal chat message. Use formatting only where it clearly helps: a comparison table, step-by-step instructions, code, formulas, a long answer with several sections, quotes, spoilers, images, maps or dates. Short answers and chit-chat get no formatting at all, not even bold words. Keep it restrained: at most a few "##"/"###" headings (never "#"), no decorative emphasis, no emoji headings, no horizontal rules between paragraphs.

Telegram Rich Markdown follows GitHub Flavored Markdown with these differences:
- Extra syntax: ||spoiler||, ==marked==, footnotes [^1], task lists, $inline LaTeX$ and $$block LaTeX$$.
- \( \) and \[ \] do NOT render as math; use $...$ and $$...$$.
- Escape literal dollar signs as \$.
- Even a spaced asterisk starts italics (3 * 4 * 5 breaks); write × or put it in backticks.
- Unknown HTML-like tags are deleted together with their content (<Name>, List<String>); put such text in backticks or write &lt;.
- Table cells only allow inline formatting.
- Citations work as described below.

Special elements:
- Dates and times: <tg-time datetime="2026-11-06T20:15:00+01:00" format="wDT">Do., 6.11., 20:15</tg-time> shows the moment in each reader's own time zone. datetime is ISO 8601 with offset (Europe/Berlin if omitted). format is "r" (relative, e.g. "in 3 hours") or any combination of w (weekday), d/D (short/long date) and t/T (short/long time), e.g. "wDT", "dt", "t". The text inside is the fallback. Use it for concrete upcoming or past moments like kick-off times, releases or deadlines.
- Collapsible sections: <details><summary>Rechenweg</summary> on its own line, then a blank line, Markdown content, a blank line and </details>. Use it for derivations, background or long lists that most readers skip; the main answer stays outside.
- Maps: <figure><tg-map lat="48.8584" long="2.2945" zoom="15"/><figcaption>Eiffelturm, Paris</figcaption></figure> on its own line. Only use coordinates returned by "geocode", never from memory. Zoom 3 (country) to 17 (building). Only add a map when the location itself matters.
- Images: <figure><img src="IMAGE_URL"/><figcaption>Short caption<cite>source domain</cite></figcaption></figure>, each image on its own line, using only Image URLs returned by "imagesearch". Group 2-3 images in a <tg-collage>, 4 or more in a <tg-slideshow> that readers swipe through: put <tg-collage> or <tg-slideshow> on its own line, a blank line, the figure lines, a blank line and the closing tag. Images and maps are always full-width blocks.
- Spoilers: ||text|| hides text until tapped, e.g. plot twists, puzzle solutions or quiz answers. Add the tg-spoiler attribute (<img src="…" tg-spoiler/>) to blur images that are graphic or reveal a spoiler.
- Quotes: <blockquote>Quoted text<cite>Person or source</cite></blockquote> for a short verbatim quote. For quotes longer than about three lines, e.g. passages from fetched pages or PDFs, use <blockquote expandable> so only the beginning shows. Inside blockquotes Markdown does not work; use <br> for line breaks and HTML tags like <b>.
- Tables: Prefer Markdown tables. Only when you need a caption, merged cells or column alignment, write one HTML table on a single line: <table bordered striped compact><caption>Title</caption><tr><th>A</th><th align="right">B</th></tr><tr><td colspan="2">…</td></tr></table>. Inside HTML tables Markdown does not work; use HTML tags like <b>.
- Buttons and user mentions via tg://user links are not available.`
)

func renderCitedMarkdown(segs []citedSegment) string {
	var sb strings.Builder
	for _, s := range segs {
		if s.url == "" {
			sb.WriteString(s.text)
			continue
		}
		_, _ = fmt.Fprintf(&sb, `<a href="%s">[%d]</a>`, utils.Escape(s.url), s.num)
	}
	return sb.String()
}

func stripRichMedia(markdown string) string {
	return richMediaLine.ReplaceAllString(markdown, "")
}

func sanitizeRich(markdown string) string {
	markdown = richButton.ReplaceAllString(markdown, "$1")
	markdown = richButtonRow.ReplaceAllString(markdown, "")
	markdown = richMarkdownUser.ReplaceAllString(markdown, "$1")
	markdown = richHTMLUser.ReplaceAllString(markdown, "$1")
	return resolveRichTimes(markdown)
}

func resolveRichTimes(markdown string) string {
	return richTimeDatetime.ReplaceAllStringFunc(markdown, func(m string) string {
		sub := richTimeDatetime.FindStringSubmatch(m)
		value := strings.TrimSpace(sub[2])
		for _, layout := range datetimeLayouts {
			if t, err := time.ParseInLocation(layout, value, berlinLocation()); err == nil {
				return fmt.Sprintf(`<tg-time %sunix="%d"%s>%s</tg-time>`, sub[1], t.Unix(), sub[3], sub[4])
			}
		}
		return sub[4]
	})
}

func buildRichMarkdown(segs []citedSegment, toolPrefix, links string) string {
	body := sanitizeRich(renderCitedMarkdown(segs))
	var sb strings.Builder
	if toolPrefix != "" {
		sb.WriteString(toolPrefix)
		if richBlockStart.MatchString(body) {
			sb.WriteString("\n\n")
		} else {
			sb.WriteString(" ")
		}
	}
	sb.WriteString(body)
	if links != "" {
		sb.WriteString("\n\n<footer>")
		sb.WriteString(links)
		sb.WriteString("</footer>")
	}
	return sb.String()
}

func embedRichMedia(markdown string, media map[string]fetchedImage) (string, []gotgbot.InputRichMessageMedia) {
	var files []gotgbot.InputRichMessageMedia
	ids := make(map[string]string)
	replace := func(re *regexp.Regexp) func(string) string {
		return func(m string) string {
			sub := re.FindStringSubmatch(m)
			src := html.UnescapeString(sub[2])
			img, ok := media[src]
			if !ok {
				return m
			}
			id, ok := ids[src]
			if !ok {
				id = fmt.Sprintf("img%d", len(ids)+1)
				ids[src] = id
				files = append(files, gotgbot.InputRichMessageMedia{
					Id: id,
					Media: gotgbot.InputMediaPhoto{
						Media: gotgbot.InputFileByReader(id, bytes.NewReader(img.data)),
					},
				})
			}
			return sub[1] + "tg://photo?id=" + id
		}
	}
	out := richImageURL.ReplaceAllStringFunc(markdown, replace(richImageURL))
	out = richImgSrc.ReplaceAllStringFunc(out, replace(richImgSrc))
	return out, files
}

type richMessageUpload struct {
	gotgbot.InputRichMessage
}

func (r richMessageUpload) Attach(_ string, w *multipart.Writer) error {
	for _, m := range r.Media {
		if err := m.Media.Attach("file_"+m.Id, w); err != nil {
			return err
		}
	}
	return nil
}

func sendRich(b *gotgbot.Bot, c plugin.GobotContext, segs []citedSegment, toolPrefix, links string, media map[string]fetchedImage) bool {
	markdown := buildRichMarkdown(segs, toolPrefix, links)
	if utf8.RuneCountInString(markdown) > MaxRichMessageLength {
		log.Warn().Int("length", utf8.RuneCountInString(markdown)).Msg("Rich message too long")
		return false
	}

	attempts := []func() gotgbot.InputRichMessage{
		func() gotgbot.InputRichMessage {
			md, files := embedRichMedia(markdown, media)
			return gotgbot.InputRichMessage{Markdown: md, Media: files}
		},
	}
	if stripped := stripRichMedia(markdown); stripped != markdown && hasRichContent(stripped, toolPrefix) {
		attempts = append(attempts, func() gotgbot.InputRichMessage {
			return gotgbot.InputRichMessage{Markdown: stripped}
		})
	}

	for _, build := range attempts {
		msg := build()
		_, err := b.Request("sendRichMessage", map[string]any{
			"chat_id":      c.EffectiveChat.Id,
			"rich_message": richMessageUpload{msg},
			"reply_parameters": gotgbot.ReplyParameters{
				MessageId:                c.EffectiveMessage.MessageId,
				AllowSendingWithoutReply: true,
			},
		}, nil)
		if err == nil {
			return true
		}
		log.Err(err).Str("markdown", msg.Markdown).Int("media", len(msg.Media)).Msg("Failed to send rich message")
	}
	return false
}

func replyContent(m *gotgbot.Message) (string, []gotgbot.PhotoSize) {
	if m.RichMessage == nil {
		return m.GetText(), m.Photo
	}
	markdown, _ := m.RichMessage.Markdown()
	return strings.TrimSpace(markdown), firstRichPhoto(m.RichMessage.Blocks)
}

// gotgbot's RichBlockChildren() returns nil for collages and slideshows, so Walk never reaches their photos.
func firstRichPhoto(blocks []gotgbot.RichBlock) []gotgbot.PhotoSize {
	for _, b := range blocks {
		var children []gotgbot.RichBlock
		switch v := b.(type) {
		case gotgbot.RichBlockPhoto:
			return v.Photo
		case gotgbot.RichBlockCollage:
			children = v.Blocks
		case gotgbot.RichBlockSlideshow:
			children = v.Blocks
		default:
			children = b.RichBlockChildren()
		}
		if photo := firstRichPhoto(children); photo != nil {
			return photo
		}
	}
	return nil
}

func hasRichContent(markdown, toolPrefix string) bool {
	return strings.TrimSpace(strings.TrimPrefix(markdown, toolPrefix)) != ""
}
