package twitter

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Brawl345/gobot/utils"
)

type (
	Response struct {
		Tweet    Tweet    `json:"data"`
		Includes Includes `json:"includes"`
	}

	Tweet struct {
		Attachments struct {
			MediaKeys []string `json:"media_keys"`
			PollIDs   []string `json:"poll_ids"`
		} `json:"attachments"`
		ID               string            `json:"id"`
		CreatedAt        time.Time         `json:"created_at"`
		AuthorID         string            `json:"author_id"`
		Text             string            `json:"text"`
		Entities         Entities          `json:"entities"`
		Withheld         Withheld          `json:"withheld"`
		ReferencedTweets []ReferencedTweet `json:"referenced_tweets"`
		PublicMetrics    PublicMetrics     `json:"public_metrics"`
	}

	Entities struct {
		URLs []URL `json:"urls"`
	}

	ReferencedTweet struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}

	Withheld struct {
		Copyright    bool     `json:"copyright"`
		CountryCodes []string `json:"country_codes"`
		Scope        string   `json:"scope"`
	}

	Includes struct {
		Media  []Media `json:"media"`
		Polls  []Poll  `json:"polls"`
		Tweets []Tweet `json:"tweets"`
		Users  []User  `json:"users"`
	}

	Media struct {
		MediaKey      string `json:"media_key"`
		Type          string `json:"type"`
		Url           string `json:"url"`
		PublicMetrics struct {
			ViewCount int `json:"view_count"`
		} `json:"public_metrics"`
	}

	Poll struct {
		ID          string    `json:"id"`
		EndDatetime time.Time `json:"end_datetime"`
		Options     []struct {
			Position int    `json:"position"`
			Label    string `json:"label"`
			Votes    int    `json:"votes"`
		} `json:"options"`
		VotingStatus string `json:"voting_status"`
	}

	PublicMetrics struct {
		RetweetCount int `json:"retweet_count"`
		ReplyCount   int `json:"reply_count"`
		LikeCount    int `json:"like_count"`
		QuoteCount   int `json:"quote_count"`
	}

	User struct {
		Verified  bool   `json:"verified"`
		ID        string `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		Protected bool   `json:"protected"`
	}

	URL struct {
		Start       int    `json:"start"`
		End         int    `json:"end"`
		Url         string `json:"url"`
		ExpandedUrl string `json:"expanded_url"`
		DisplayUrl  string `json:"display_url"`
	}

	Response11 struct {
		ExtendedEntities struct {
			Media []Media11 `json:"media"`
		} `json:"extended_entities"`
	}

	Media11 struct {
		VideoInfo VideoInfo `json:"video_info"`
	}

	VideoInfo struct {
		Variants []struct {
			Bitrate     int    `json:"bitrate"`
			ContentType string `json:"content_type"`
			Url         string `json:"url"`
		} `json:"variants"`
	}

	Error struct {
		Errors []struct {
			Parameters map[string][]string `json:"parameters"`
			Message    string              `json:"message"`
		} `json:"errors"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
		Type   string `json:"type"`
	}

	PartialError struct {
		Errors []struct {
			Type   string `json:"type"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
)

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Title, e.Detail)
}

func (e *PartialError) Error() string {
	var sb strings.Builder
	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("[%s: %s - %s]", err.Title, err.Detail, err.Type))
		if i != len(e.Errors)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}

func (r *Response) Quote() *Tweet {
	var quotedId string
	for _, refTweet := range r.Tweet.ReferencedTweets {
		if refTweet.Type == "quoted" {
			quotedId = refTweet.ID
		}
	}

	if quotedId == "" {
		return nil
	}

	for _, tweet := range r.Includes.Tweets {
		if tweet.ID == quotedId {
			return &tweet
		}
	}

	return nil
}

func (i *Includes) User(userId string) *User {
	for _, user := range i.Users {
		if user.ID == userId {
			return &user
		}
	}
	return nil
}

func (u *User) String() string {
	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b> (<a href=\"https://twitter.com/%s\">@%s</a>",
			html.EscapeString(u.Name),
			u.Username,
			u.Username,
		),
	)

	if u.Verified {
		sb.WriteString(" ✅")
	}

	if u.Protected {
		sb.WriteString(" 🔒")
	}

	sb.WriteString("):")

	return sb.String()
}

func (w *Withheld) InGermany() bool {
	for _, countryCode := range w.CountryCodes {
		if countryCode == "DE" {
			return true
		}
	}
	return false
}

func (w *Withheld) String() string {
	var sb strings.Builder
	sb.WriteString("<i>❌ Dieser Tweet ist aufgrund")

	if w.Copyright {
		sb.WriteString(" eines Urheberrechtsverstoßes")
	} else {
		sb.WriteString(" von lokalen Gesetzen")
	}

	sb.WriteString(" in Deutschland nicht verfügbar.</i>")
	return sb.String()
}

func (p *Poll) Closed() bool {
	return p.VotingStatus == "closed"
}

func (p *Poll) TotalVotes() int {
	var total int
	for _, option := range p.Options {
		total += option.Votes
	}
	return total
}

func (p *PublicMetrics) String() string {
	var sb strings.Builder

	if p.RetweetCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 🔁 %s",
				utils.FormatThousand(p.RetweetCount),
			),
		)
	}

	if p.QuoteCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 💬 %s",
				utils.FormatThousand(p.QuoteCount),
			),
		)
	}

	if p.LikeCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | ❤ %s",
				utils.FormatThousand(p.LikeCount),
			),
		)
	}

	return sb.String()
}

// HighestResolution returns the URL of the video with the highest bitrate/resolution
func (m *Media11) HighestResolution() string {
	var bitrate int
	var index int
	for i, variant := range m.VideoInfo.Variants {
		if variant.Bitrate > bitrate {
			bitrate = variant.Bitrate
			index = i
		}
	}

	return m.VideoInfo.Variants[index].Url
}
