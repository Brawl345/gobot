package wikipedia

import "regexp"

var (
	regexDisambiguation = regexp.MustCompile(`(?iUm)<li>(.+)</li>`)
	regexHTML           = regexp.MustCompile("<.*?>")
	regexSection        = regexp.MustCompile(`\n+=+ (.+) =+\n+(.*)`)
	regexWprov          = regexp.MustCompile(`\?wprov=.*`)
)

type (
	Response struct {
		Query struct {
			Pages []struct {
				Title         string    `json:"title"`
				URL           string    `json:"fullurl"`
				Text          string    `json:"extract"`
				Pageprops     PageProps `json:"pageprops"`
				Missing       bool      `json:"missing"`
				Invalid       bool      `json:"invalid"`
				InvalidReason string    `json:"invalidreason"`
			} `json:"pages"`
		} `json:"query"`
	}

	PageProps struct {
		Disambiguation bool `json:"disambiguation"`
	}
)

func (p *PageProps) UnmarshalJSON([]byte) error {
	// if pageprops is defined, it's always a disambiguation page
	p.Disambiguation = true
	return nil
}
