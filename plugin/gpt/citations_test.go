package gpt

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func cite(ids ...string) string {
	return citationStart + "cite" + citationDelimiter + strings.Join(ids, citationDelimiter) + citationStop
}

func TestResolveCitations(t *testing.T) {
	c := &Citations{sources: map[string]string{
		"turn1search0": "https://a.example/?x=1&y=2",
		"turn1search1": "https://b.example/",
		"turn2view0":   "https://a.example/?x=1&y=2",
	}}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single citation after punctuation",
			in:   "Satz eins. " + cite("turn1search0") + "\nSatz zwei.",
			want: `Satz eins. <a href="https://a.example/?x=1&y=2">[1]</a>` + "\nSatz zwei.",
		},
		{
			name: "multiple sources deduplicated by URL",
			in:   "A < B. " + cite("turn1search1", "turn1search0", "turn2view0"),
			want: `A &lt; B. <a href="https://b.example/">[1]</a><a href="https://a.example/?x=1&y=2">[2]</a>`,
		},
		{
			name: "number reused for repeated source",
			in:   "Eins." + cite("turn1search0") + " Zwei. " + cite("turn2view0"),
			want: `Eins. <a href="https://a.example/?x=1&y=2">[1]</a> Zwei. <a href="https://a.example/?x=1&y=2">[1]</a>`,
		},
		{
			name: "line locator ignored",
			in:   "Text. " + cite("turn1search1", "L3-L5"),
			want: `Text. <a href="https://b.example/">[1]</a>`,
		},
		{
			name: "unknown ID and foreign family dropped",
			in:   "Nie floaten. " + cite("turn680151view0") + " Mehr " + citationStart + "filecite" + citationDelimiter + "turn0file1" + citationStop + "Text.",
			want: "Nie floaten. MehrText.",
		},
		{
			name: "entity replaced by name",
			in:   "Firma " + citationStart + "entity" + citationDelimiter + `["company","Tektronix",0]` + citationStop + " baut Sonden.",
			want: "Firma Tektronix baut Sonden.",
		},
		{
			name: "unterminated marker at end",
			in:   "Abgeschnitten. " + citationStart + "cite" + citationDelimiter + "turn1sea",
			want: "Abgeschnitten.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderCitedHTML(c.Resolve(tt.in), 4096); got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestRenderCitedHTMLTruncates(t *testing.T) {
	c := &Citations{sources: map[string]string{"turn1search0": "https://a.example/"}}
	segs := c.Resolve(strings.Repeat("<", 7) + " " + cite("turn1search0") + strings.Repeat("x", 50))

	got := renderCitedHTML(segs, 30)
	if n := utf8.RuneCountInString(got); n > 30 {
		t.Fatalf("length %d exceeds budget: %q", n, got)
	}
	if !strings.HasSuffix(got, "...") || strings.Contains(got, "<a") || strings.Count(got, "&lt;") != 6 || strings.Contains(got, "&l...") {
		t.Fatalf("bad truncation: %q", got)
	}
}

func TestCitationsAddAndTurns(t *testing.T) {
	c := NewCitations()
	first, second := c.NextTurn(), c.NextTurn()
	if second != first+1 {
		t.Fatalf("turns not sequential: %d, %d", first, second)
	}
	marker := c.Add("turn5view0", "https://x.example/")
	if marker != cite("turn5view0") {
		t.Fatalf("unexpected marker %q", marker)
	}
	if !hasCitationLinks(c.Resolve("Text. " + marker)) {
		t.Fatal("registered marker did not resolve")
	}
}
