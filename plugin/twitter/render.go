package twitter

import (
	"cmp"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

const collapseTextLength = 600

var (
	tcoRegex     = regexp.MustCompile(`https://t\.co/\w+`)
	mentionRegex = regexp.MustCompile(`(^|[^\w])@(\w{1,15})`)
	hashtagRegex = regexp.MustCompile(`(^|[^\w&])#(\p{L}[\p{L}\p{N}_]*)`)
)

type tweetMessage struct {
	html    strings.Builder
	media   []gotgbot.InputRichMessageMedia
	uploads int
}

func renderTweet(result Result, prepared []*preparedMedium, withMedia, spoiler bool) *tweetMessage {
	msg := &tweetMessage{}
	link := statusLink(result.Core.UserResults.Result.Core.ScreenName, result.RestId)

	msg.html.WriteString("<p>" + header(result.Core.UserResults, result.Legacy) + "</p>")
	msg.html.WriteString(paragraph(tweetText(result.NoteTweet, result.Legacy, result.Article.ArticleResults.Result.Title)))
	if withMedia {
		msg.addMedia(prepared, spoiler)
	}
	msg.html.WriteString(pollTable(result.Card))
	msg.html.WriteString(communityNote(result.BirdwatchPivot))
	msg.html.WriteString(quote(result))
	msg.html.WriteString("<footer>" + footer(result.Legacy, link) + "</footer>")
	return msg
}

func statusLink(screenName, id string) string {
	return fmt.Sprintf("https://x.com/%s/status/%s", screenName, id)
}

func header(user UserResult, legacy Legacy) string {
	h := user.Author()
	if legacy.InReplyToScreenName != "" {
		h += fmt.Sprintf(
			` als Antwort an <a href="%s">@%s</a>`,
			statusLink(legacy.InReplyToScreenName, legacy.InReplyToStatusIdStr),
			utils.Escape(legacy.InReplyToScreenName),
		)
	}
	return h + ":"
}

func tweetText(note NoteTweet, legacy Legacy, articleTitle string) string {
	text := note.NoteTweetResults.Result.Text
	urls := make(map[string]Url)
	if text != "" {
		for _, e := range note.NoteTweetResults.Result.EntitySet.Urls {
			urls[e.Url] = Url{DisplayUrl: e.DisplayUrl, ExpandedUrl: e.ExpandedUrl, Url: e.Url}
		}
	} else {
		text = legacy.FullText
		for _, e := range legacy.Entities.Urls {
			urls[e.Url] = e
		}
	}
	media := make(map[string]bool)
	for _, m := range legacy.ExtendedEntities.Media {
		media[m.Url] = true
	}

	var sb strings.Builder
	last := 0
	for _, loc := range tcoRegex.FindAllStringIndex(text, -1) {
		sb.WriteString(escapeText(text[last:loc[0]]))
		last = loc[1]
		tco := text[loc[0]:loc[1]]
		if media[tco] {
			continue
		}
		u, ok := urls[tco]
		if !ok {
			u = Url{DisplayUrl: tco, ExpandedUrl: tco}
		}
		if articleTitle != "" && strings.Contains(u.ExpandedUrl, "/i/article/") {
			u.DisplayUrl = "📰 " + articleTitle
		}
		sb.WriteString(fmt.Sprintf(
			`<a href="%s">%s</a>`,
			utils.Escape(u.ExpandedUrl),
			utils.Escape(cmp.Or(u.DisplayUrl, u.ExpandedUrl)),
		))
	}
	sb.WriteString(escapeText(text[last:]))
	return lineBreaks(strings.TrimSpace(sb.String()))
}

func escapeText(s string) string {
	s = utils.Escape(s)
	s = mentionRegex.ReplaceAllString(s, `$1<a href="https://x.com/$2">@$2</a>`)
	s = hashtagRegex.ReplaceAllString(s, `$1<a href="https://x.com/hashtag/$2">#$2</a>`)
	return s
}

func lineBreaks(s string) string {
	return strings.ReplaceAll(s, "\n", "<br>")
}

func paragraph(html string) string {
	if html == "" {
		return ""
	}
	if len([]rune(html)) > collapseTextLength {
		return "<blockquote expandable>" + html + "</blockquote>"
	}
	return "<p>" + html + "</p>"
}

func (msg *tweetMessage) addMedia(prepared []*preparedMedium, spoiler bool) {
	spoilerAttr := ""
	if spoiler {
		spoilerAttr = " tg-spoiler"
	}

	var tags, fullLinks, missingLinks []string
	for _, p := range prepared {
		if !p.usable() {
			missingLinks = append(missingLinks, mediumLink(p, cmpLink(p), len(prepared)))
			continue
		}

		id := fmt.Sprintf("m%d", len(msg.media)+1)
		file := p.inputFile(id)
		if p.upload {
			msg.uploads++
		}

		switch {
		case p.medium.IsPhoto():
			msg.media = append(msg.media, gotgbot.InputRichMessageMedia{Id: id, Media: gotgbot.InputMediaPhoto{Media: file}})
			tags = append(tags, fmt.Sprintf(`<img src="tg://photo?id=%s"%s/>`, id, spoilerAttr))
		case p.medium.IsGIF():
			msg.media = append(msg.media, gotgbot.InputRichMessageMedia{Id: id, Media: gotgbot.InputMediaAnimation{Media: file}})
			tags = append(tags, fmt.Sprintf(`<video src="tg://video?id=%s"%s></video>`, id, spoilerAttr))
		default:
			msg.media = append(msg.media, gotgbot.InputRichMessageMedia{Id: id, Media: gotgbot.InputMediaVideo{Media: file, SupportsStreaming: true}})
			tags = append(tags, fmt.Sprintf(`<video src="tg://video?id=%s"%s></video>`, id, spoilerAttr))
		}

		if p.fullURL != "" {
			fullLinks = append(fullLinks, mediumLink(p, p.fullURL, len(prepared)))
		}
	}

	switch len(tags) {
	case 0:
	case 1:
		msg.html.WriteString(tags[0])
	default:
		msg.html.WriteString("<tg-collage>" + strings.Join(tags, "") + "</tg-collage>")
	}

	var lines []string
	if len(fullLinks) > 0 {
		lines = append(lines, "🔍 Volle Auflösung: "+strings.Join(fullLinks, " · "))
	}
	if len(missingLinks) > 0 {
		lines = append(lines, "⚠️ Nicht ladbar: "+strings.Join(missingLinks, " · "))
	}
	if len(lines) > 0 {
		msg.html.WriteString("<footer>" + strings.Join(lines, "<br>") + "</footer>")
	}
}

func mediumLink(p *preparedMedium, link string, total int) string {
	label := "Bild"
	if !p.medium.IsPhoto() {
		label = "Video"
	}
	if total > 1 {
		label += fmt.Sprintf(" %d", p.position)
	}
	if p.fullSize > 0 {
		label += fmt.Sprintf(" (%s MB)", utils.FormatThousand(p.fullSize/1_000_000))
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, utils.Escape(link), label)
}

func footer(legacy Legacy, link string) string {
	createdAt, err := time.Parse(time.RubyDate, legacy.CreatedAt)
	date := utils.Escape(legacy.CreatedAt)
	if err == nil {
		date = createdAt.In(utils.GermanTimezone()).Format("02.01.2006, 15:04:05 Uhr")
	}
	return fmt.Sprintf(`📅 <a href="%s">%s</a>%s`, utils.Escape(link), date, legacy.Metrics())
}

func pollTable(card Card) string {
	if !card.HasPoll() {
		return ""
	}
	poll, err := card.Poll()
	if err != nil {
		log.Err(err).Msg("Failed to parse poll")
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<table bordered striped compact><caption>📊 Umfrage")
	if poll.Closed() {
		sb.WriteString(" (beendet)")
	}
	sb.WriteString("</caption>")
	for _, option := range poll.Options {
		percentage := float64(option.Votes) / float64(poll.TotalVotes) * 100
		sb.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td align="right">%s</td><td align="right">%.1f %%</td></tr>`,
			utils.Escape(option.Label),
			utils.FormatThousand(option.Votes),
			percentage,
		))
	}
	sb.WriteString("</table>")

	ends := "endet"
	if poll.Closed() {
		ends = "endete"
	}
	sb.WriteString(fmt.Sprintf(
		"<footer>%s · %s am %s</footer>",
		votes(poll.TotalVotes),
		ends,
		poll.EndDatetime.In(utils.GermanTimezone()).Format("02.01.2006, 15:04:05 Uhr"),
	))
	return sb.String()
}

func votes(n int) string {
	if n == 1 {
		return "1 Stimme"
	}
	return utils.FormatThousand(n) + " Stimmen"
}

func communityNote(pivot BirdwatchPivot) string {
	if pivot.DestinationUrl == "" {
		return ""
	}
	html := fmt.Sprintf(
		`<blockquote expandable><b>⚠️ <a href="%s">Leser haben Kontext hinzugefügt</a></b>`,
		utils.Escape(pivot.DestinationUrl),
	)
	if text := birdwatchText(pivot.Subtitle); text != "" {
		html += "<br>" + text
	}
	return html + "</blockquote>"
}

func birdwatchText(t BirdwatchText) string {
	units := utf16.Encode([]rune(t.Text))
	var sb strings.Builder
	last := 0
	for _, e := range t.Entities {
		if e.FromIndex < last || e.ToIndex > len(units) || e.FromIndex >= e.ToIndex {
			continue
		}
		sb.WriteString(utils.Escape(string(utf16.Decode(units[last:e.FromIndex]))))
		sb.WriteString(fmt.Sprintf(
			`<a href="%s">%s</a>`,
			utils.Escape(e.Ref.Url),
			utils.Escape(string(utf16.Decode(units[e.FromIndex:e.ToIndex]))),
		))
		last = e.ToIndex
	}
	sb.WriteString(utils.Escape(string(utf16.Decode(units[last:]))))
	return lineBreaks(strings.TrimSpace(sb.String()))
}

func mediaHint(media []Medium, link string) string {
	var photos, videos, gifs int
	for _, m := range media {
		switch {
		case m.IsPhoto():
			photos++
		case m.IsGIF():
			gifs++
		default:
			videos++
		}
	}

	count := func(n int, one, many string) string {
		if n == 1 {
			return "1 " + one
		}
		return fmt.Sprintf("%d %s", n, many)
	}
	var parts []string
	if photos > 0 {
		parts = append(parts, "🖼 "+count(photos, "Bild", "Bilder"))
	}
	if videos > 0 {
		parts = append(parts, "🎬 "+count(videos, "Video", "Videos"))
	}
	if gifs > 0 {
		parts = append(parts, "🎞 "+count(gifs, "GIF", "GIFs"))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, utils.Escape(link), strings.Join(parts, ", "))
}

func quote(result Result) string {
	q := result.QuotedStatusResult.Result
	var author, cite string
	var body []string

	switch q.Typename {
	case "Tweet", "TweetWithVisibilityResults":
		qs := q.Tweet
		if q.TweetSub.RestId != "" {
			qs = q.TweetSub
		}
		link := statusLink(qs.Core.UserResults.Result.Core.ScreenName, qs.RestId)
		author = header(qs.Core.UserResults, qs.Legacy)
		if text := tweetText(qs.NoteTweet, qs.Legacy, ""); text != "" {
			body = append(body, text)
		}
		if hint := mediaHint(qs.Legacy.ExtendedEntities.Media, link); hint != "" {
			body = append(body, hint)
		}
		if qs.Card.HasPoll() {
			poll, err := qs.Card.Poll()
			if err != nil {
				log.Err(err).Str("tweetID", qs.RestId).Msg("Failed to parse quote poll")
			} else {
				body = append(body, lineBreaks(strings.TrimSpace(pollText(poll))))
			}
		}
		cite = footer(qs.Legacy, link)
	case "TweetTombstone", "TweetUnavailable":
		user := q.Tombstone.UserResults
		link := statusLink("i", result.Legacy.QuotedStatusIdStr)
		if user.Result.Core.ScreenName != "" {
			author = user.Author() + ":"
			link = statusLink(user.Result.Core.ScreenName, result.Legacy.QuotedStatusIdStr)
		}
		switch {
		case q.Tombstone.Typename == "BlurredMediaTombstone" || q.Reason == "NsfwLoggedOut" || q.Reason == "NsfwViewerHasNoStatedAge":
			body = append(body, fmt.Sprintf(`<i>🔞 <a href="%s">Nicht jugendfreier Inhalt</a></i>`, utils.Escape(link)))
		case q.Reason == "Protected":
			body = append(body, "<i>🔓 Der Account-Inhaber hat beschränkt, wer seine Tweets ansehen kann.</i>")
		default:
			body = append(body, fmt.Sprintf(`<i>❌ <a href="%s">Zitat nicht einsehbar</a></i>`, utils.Escape(link)))
		}
	default:
		return ""
	}

	heading := "<b>Zitat</b>"
	if author != "" {
		heading = "<b>Zitat von</b> " + author
	}

	html := "<blockquote expandable>" + heading
	for _, part := range body {
		html += "<br>" + part
	}
	if cite != "" {
		html += "<cite>" + cite + "</cite>"
	}
	return html + "</blockquote>"
}
