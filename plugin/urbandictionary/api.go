package urbandictionary

import "time"

const Url = "https://api.urbandictionary.com/v0/define?term=%s"

type (
	Response struct {
		List []Term `json:"list"`
	}

	Term struct {
		Permalink  string    `json:"permalink"`
		Definition string    `json:"definition"`
		Word       string    `json:"word"`
		Example    string    `json:"example"`
		Upvotes    int       `json:"thumbs_up"`
		Downvotes  int       `json:"thumbs_down"`
		WrittenOn  time.Time `json:"written_on"`
	}
)
