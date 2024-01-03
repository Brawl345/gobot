package youtube

import (
	"errors"
	"time"

	"github.com/sosodev/duration"
	"golang.org/x/exp/slices"
)

var ErrNoVideoFound = errors.New("no video found")

type (
	Response struct {
		Items []Video `json:"items"`
	}

	SearchResponse struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
		} `json:"items"`
	}

	Video struct {
		ID      string `json:"id"`
		Snippet struct {
			PublishedAt  time.Time `json:"publishedAt"`
			ChannelID    string    `json:"channelId"`
			Title        string    `json:"title"`
			ChannelTitle string    `json:"channelTitle"`
		} `json:"snippet"`
		ContentDetails ContentDetails `json:"contentDetails"`
		Statistics     struct {
			ViewCount    uint64 `json:"viewCount,string"`
			LikeCount    uint64 `json:"likeCount,string"`
			CommentCount uint64 `json:"commentCount,string"`
		} `json:"statistics"`
		LiveStreamingDetails struct {
			ActualStartTime    time.Time `json:"actualStartTime"`
			ActualEndTime      time.Time `json:"actualEndTime"`
			ConcurrentViewers  uint64    `json:"concurrentViewers,string"`
			ScheduledStartTime time.Time `json:"scheduledStartTime"`
			ScheduledEndTime   time.Time `json:"scheduledEndTime"`
		} `json:"liveStreamingDetails"`
	}

	ContentDetails struct {
		Duration          string `json:"duration"`
		RegionRestriction struct {
			Allowed []string `json:"allowed"`
			Blocked []string `json:"blocked"`
		} `json:"regionRestriction"`
	}

	DeArrowResponse struct {
		Titles []struct {
			Title    string `json:"title"`
			Original bool   `json:"original"`
			Votes    int    `json:"votes"`
			Locked   bool   `json:"locked"`
		} `json:"titles"`
	}
)

func (v *Video) BlockedInGermany() bool {
	if slices.Contains(v.ContentDetails.RegionRestriction.Blocked, "DE") {
		return true
	}

	if len(v.ContentDetails.RegionRestriction.Allowed) > 0 &&
		!slices.Contains(v.ContentDetails.RegionRestriction.Allowed, "DE") {
		return true
	}

	return false
}

func (v *Video) IsPremiere() bool {
	return v.IsLive() && v.ContentDetails.Duration != "P0D"
}

func (v *Video) IsScheduledLive() bool {
	return v.IsLive() && !v.IsLiveNow() && !v.WasLive()
}

func (v *Video) IsLive() bool {
	return !v.LiveStreamingDetails.ActualStartTime.IsZero() || !v.LiveStreamingDetails.ScheduledStartTime.IsZero()
}

func (v *Video) IsLiveNow() bool {
	return !v.LiveStreamingDetails.ActualStartTime.IsZero() && v.LiveStreamingDetails.ActualEndTime.IsZero()
}

func (v *Video) WasLive() bool {
	return !v.LiveStreamingDetails.ActualEndTime.IsZero()
}

func (c *ContentDetails) ParseDuration() (*duration.Duration, error) {
	return duration.Parse(c.Duration)
}

func (d *DeArrowResponse) GetBestTitle() string {
	// From the API docs:
	// "Data is returned ordered. You can use the first element. However, you should make sure the first element
	// has either locked = true or votes >= 0. If not, it is considered untrusted and is only to be shown
	// in the voting box until it has been confirmed by another user."

	if len(d.Titles) == 0 {
		return ""
	}

	if d.Titles[0].Locked && !d.Titles[0].Original {
		return d.Titles[0].Title
	}

	// Will check for the highest number of votes instead of just taking the first one.
	// API docs say that you should not use titles without votes but the browser extension does it anyway.
	// So we will do it too!
	maxVotes := -1
	var title string
	for _, t := range d.Titles {
		t := t
		if t.Votes > maxVotes && !t.Original {
			maxVotes = t.Votes
			title = t.Title
		}
	}

	return title
}
