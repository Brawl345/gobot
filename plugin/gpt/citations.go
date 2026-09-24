package gpt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Brawl345/gobot/utils"
)

// Private Use characters U+E200, U+E202 and U+E201 delimit citation markers.
const (
	citationStart     = "\xee\x88\x80"
	citationDelimiter = "\xee\x88\x82"
	citationStop      = "\xee\x88\x81"
)

var (
	citationRegex     = regexp.MustCompile(`[ \t]*\x{E200}([^\x{E200}\x{E201}]*)(?:\x{E201}|$)`)
	strayCitationChar = regexp.MustCompile(`[\x{E200}-\x{E202}]`)
	lineLocatorRegex  = regexp.MustCompile(`^L\d+(?:-L\d+)?$`)

	CitationInstruction = strings.NewReplacer(
		"{CITATION_START}", citationStart,
		"{CITATION_DELIMITER}", citationDelimiter,
		"{CITATION_STOP}", citationStop,
	).Replace(`## Citations

Results are returned by "websearch" and "webfetch". Each search result and each fetched page is called a "source" and identified by its reference ID, which is the ID inside its "Citation Marker" line (e.g. turn2search1 or turn3view0).

Citations are references to "websearch" and "webfetch" sources. Citations may be used to refer to either a single source or multiple sources.

Citations to a single source must be written as {CITATION_START}cite{CITATION_DELIMITER}turn\d+\w+\d+{CITATION_STOP} (e.g. {CITATION_START}cite{CITATION_DELIMITER}turn2search5{CITATION_STOP}).

Citations to multiple sources must be written as {CITATION_START}cite{CITATION_DELIMITER}turn\d+\w+\d+{CITATION_DELIMITER}turn\d+\w+\d+{CITATION_DELIMITER}...{CITATION_STOP} (e.g. {CITATION_START}cite{CITATION_DELIMITER}turn2search5{CITATION_DELIMITER}turn3view0{CITATION_DELIMITER}...{CITATION_STOP}).

Citations must not be placed inside markdown bold, italics, or code fences, as they will not display correctly. Instead, place the citations outside the markdown block. Citations outside code fences may not be placed on the same line as the end of the code fence.

You must NOT write reference ID turn\d+\w+\d+ verbatim in the response text without putting them between {CITATION_START}...{CITATION_STOP}.

- Place citations at the end of the paragraph, or inline if the paragraph is long, unless the user requests specific citation placement.
- Citations must be placed after punctuation.
- Citations must not be all grouped together at the end of the response.
- Citations must not be put in a line or paragraph with nothing else but the citations themselves.
- Never invent reference IDs that were not returned by a tool.
- Do not attempt to cite items without a corresponding citation marker, as they are not meant to be cited.

<extra_considerations_for_citations>
- **Relevance:** Include only search results and citations that support the cited response text. Irrelevant sources permanently degrade user trust.
- **Diversity:** Where the results allow it, base your answer on sources from diverse domains, and cite accordingly.
- **Trustworthiness:** To produce a credible response, you must rely on high quality domains, and ignore information from less reputable domains unless they are the only source.
- **Accurate Representation:** Each citation must accurately reflect the source content. Selective interpretation of the source content is not allowed.

Remember, the quality of a domain/source depends on the context.
- When multiple viewpoints exist, cite sources covering the spectrum of opinions to ensure balance and comprehensiveness.
- When reliable sources disagree, cite at least one high-quality source for each major viewpoint.
- Ensure more than half of citations come from widely recognized authoritative outlets on the topic.
- For debated topics, cite at least one reliable source representing each major viewpoint.
- Do not ignore the content of a relevant source because it is low quality.
</extra_considerations_for_citations>`)
)

type (
	// Citations maps the source IDs handed to the model in tool outputs to URLs.
	Citations struct {
		mu      sync.Mutex
		turn    int
		sources map[string]string
	}

	// citedSegment is either plain text or a numbered link to a cited source.
	citedSegment struct {
		text string
		url  string
		num  int
	}
)

func NewCitations() *Citations {
	// Turn numbers stay unique within a stored conversation, so markers the
	// model repeats from earlier requests never resolve to current sources.
	return &Citations{
		turn:    int(time.Now().Unix()%100_000) * 100,
		sources: make(map[string]string),
	}
}

func (c *Citations) NextTurn() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	turn := c.turn
	c.turn++
	return turn
}

// Add registers a source and returns its citation marker for the tool output.
func (c *Citations) Add(id, url string) string {
	c.mu.Lock()
	c.sources[id] = url
	c.mu.Unlock()
	return citationStart + "cite" + citationDelimiter + id + citationStop
}

// stripCitationChars removes marker delimiters from external content so it
// cannot fake citation markers in tool outputs.
func stripCitationChars(s string) string {
	return strayCitationChar.ReplaceAllString(s, "")
}

func (c *Citations) lookup(id string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	url, ok := c.sources[id]
	return url, ok
}

// Resolve splits model output at its citation markers. Sources are numbered by
// first appearance, unknown IDs and other marker families are dropped, and
// entity markers are replaced by their display name.
func (c *Citations) Resolve(text string) []citedSegment {
	var segs []citedSegment
	numbers := make(map[string]int)

	appendText := func(s string) {
		s = stripCitationChars(s)
		if s == "" {
			return
		}
		if n := len(segs); n > 0 && segs[n-1].url == "" {
			segs[n-1].text += s
			return
		}
		segs = append(segs, citedSegment{text: s})
	}

	last := 0
	for _, m := range citationRegex.FindAllStringSubmatchIndex(text, -1) {
		appendText(text[last:m[0]])
		last = m[1]
		leadingSpace := text[m[0] : m[2]-len(citationStart)]

		parts := strings.Split(text[m[2]:m[3]], citationDelimiter)
		switch parts[0] {
		case "cite":
		case "entity":
			if len(parts) > 1 {
				var entity []any
				if json.Unmarshal([]byte(parts[1]), &entity) == nil && len(entity) > 1 {
					if name, ok := entity[1].(string); ok {
						appendText(leadingSpace + name)
					}
				}
			}
			continue
		default:
			continue
		}

		seen := make(map[string]struct{})
		for _, id := range parts[1:] {
			id = strings.TrimSpace(id)
			if lineLocatorRegex.MatchString(id) {
				continue
			}
			url, ok := c.lookup(id)
			if !ok {
				continue
			}
			if _, dup := seen[url]; dup {
				continue
			}
			seen[url] = struct{}{}
			num, ok := numbers[url]
			if !ok {
				num = len(numbers) + 1
				numbers[url] = num
			}
			if n := len(segs); len(seen) == 1 && n > 0 && segs[n-1].url == "" && !strings.HasSuffix(segs[n-1].text, "\n") {
				segs[n-1].text += " "
			}
			segs = append(segs, citedSegment{url: url, num: num})
		}
	}
	appendText(text[last:])

	if len(segs) > 0 && segs[0].url == "" {
		segs[0].text = strings.TrimLeft(segs[0].text, " \t\n")
	}
	if n := len(segs); n > 0 && segs[n-1].url == "" {
		segs[n-1].text = strings.TrimRight(segs[n-1].text, " \t\n")
	}
	return segs
}

func (s citedSegment) html() string {
	if s.url == "" {
		return utils.Escape(s.text)
	}
	return fmt.Sprintf(`<a href="%s">[%d]</a>`, utils.Escape(s.url), s.num)
}

func hasCitationLinks(segs []citedSegment) bool {
	for _, s := range segs {
		if s.url != "" {
			return true
		}
	}
	return false
}

func plainText(segs []citedSegment) string {
	var sb strings.Builder
	for _, s := range segs {
		sb.WriteString(s.text)
	}
	return sb.String()
}

// renderCitedHTML renders the segments as Telegram HTML within budget runes.
// Truncation happens on raw runes, so it never cuts inside an entity or link.
func renderCitedHTML(segs []citedSegment, budget int) string {
	var full strings.Builder
	for _, s := range segs {
		full.WriteString(s.html())
	}
	if utf8.RuneCountInString(full.String()) <= budget {
		return full.String()
	}

	const ellipsis = "..."
	budget -= len(ellipsis)
	var sb strings.Builder
	used := 0
outer:
	for _, s := range segs {
		if s.url != "" {
			link := s.html()
			n := utf8.RuneCountInString(link)
			if used+n > budget {
				break
			}
			sb.WriteString(link)
			used += n
			continue
		}
		for _, r := range s.text {
			escaped := utils.Escape(string(r))
			n := utf8.RuneCountInString(escaped)
			if used+n > budget {
				break outer
			}
			sb.WriteString(escaped)
			used += n
		}
	}
	return sb.String() + ellipsis
}
