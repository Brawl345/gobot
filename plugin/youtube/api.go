package youtube

import (
	"time"

	"github.com/sosodev/duration"
	"golang.org/x/exp/slices"
)

type (
	Response struct {
		Items []Video `json:"items"`
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
