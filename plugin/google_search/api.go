package google_search

type Response struct {
	Queries struct {
		Request []struct {
			SearchTerms string `json:"searchTerms"`
		} `json:"request"`
	} `json:"queries"`
	SearchInformation struct {
		FormattedTotalResults string `json:"formattedTotalResults"`
	} `json:"searchInformation"`
	Items []struct {
		Title       string `json:"title"`
		Link        string `json:"link"`
		DisplayLink string `json:"displayLink"`
	} `json:"items"`
}
