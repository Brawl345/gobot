package youtube

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("youtube")

type Plugin struct {
	apiKey string
}

func New(credentialService models.CredentialService) *Plugin {
	apiKey, err := credentialService.GetKey("google_api_key")
	if err != nil {
		log.Warn().Msg("google_api_key not found")
	}

	return &Plugin{
		apiKey: apiKey,
	}
}

func (p *Plugin) Name() string {
	return "youtube"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "yt",
			Description: "<Suchbegriff> - Auf YouTube suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	// For videoId see https://webapps.stackexchange.com/a/101153
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)youtube\.com/watch(?:\?|\?.+&)?v=([\dA-Za-z_-]{10}[048AEIMQUYcgkosw])`),
			HandlerFunc: p.OnYouTubeLink,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)youtube\.com/(?:embed|shorts)/([\dA-Za-z_-]{10}[048AEIMQUYcgkosw])`),
			HandlerFunc: p.OnYouTubeLink,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)youtu\.be/([\dA-Za-z_-]{10}[048AEIMQUYcgkosw])`),
			HandlerFunc: p.OnYouTubeLink,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/yt(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onYouTubeSearch,
		},
	}
}

func (p *Plugin) getVideoInfo(videoID string) (Video, error) {
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/youtube/v3/videos",
	}

	q := requestUrl.Query()
	q.Set("key", p.apiKey)
	q.Set("id", videoID)
	q.Set("part", "snippet,statistics,contentDetails,liveStreamingDetails")
	q.Set(
		"fields",
		"items/id,items/snippet(publishedAt,channelId,channelTitle,title),"+
			"items/statistics(viewCount,likeCount,commentCount),"+
			"items/contentDetails(duration,regionRestriction),"+
			"items/liveStreamingDetails(scheduledStartTime,scheduledEndTime,actualStartTime,actualEndTime,concurrentViewers)",
	)

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := utils.GetRequest(requestUrl.String(), &response)

	if err != nil {
		return Video{}, err
	}

	if len(response.Items) == 0 {
		return Video{}, ErrNoVideoFound
	}

	return response.Items[0], nil
}

func constructText(video *Video) string {
	var sb strings.Builder

	// Title
	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b>\n",
			html.EscapeString(video.Snippet.Title),
		),
	)

	// Uploader
	sb.WriteString(
		fmt.Sprintf(
			"🎥 <strong><a href=\"https://www.youtube.com/channel/%s/videos\">%s</a></strong>",
			video.Snippet.ChannelID,
			html.EscapeString(video.Snippet.ChannelTitle),
		),
	)

	// Uploaded at
	timezone := utils.GermanTimezone()
	sb.WriteString(
		fmt.Sprintf(
			" | 📅 %s\n",
			video.Snippet.PublishedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
		),
	)

	// Scheduled livestream
	if video.IsScheduledLive() {
		videoType := "Livestream"
		if video.IsPremiere() {
			videoType = "Premiere"
		}

		sb.WriteString(
			fmt.Sprintf(
				"🔴 %s startet am %s",
				videoType,
				video.LiveStreamingDetails.ScheduledStartTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)

		// Livestream scheduled until
		if !video.LiveStreamingDetails.ScheduledEndTime.IsZero() {
			sb.WriteString(
				fmt.Sprintf(
					" und endet voraussichtlich am %s",
					video.LiveStreamingDetails.ScheduledEndTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
				),
			)
		}

		sb.WriteString("\n")
	}

	// Livestream is currently running
	if video.IsLiveNow() {
		sb.WriteString(
			fmt.Sprintf(
				"🔴 Live seit %s",
				video.LiveStreamingDetails.ActualStartTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)

		// Livestream runs until
		if !video.LiveStreamingDetails.ScheduledEndTime.IsZero() {
			sb.WriteString(
				fmt.Sprintf(
					" bis voraussichtlich %s",
					video.LiveStreamingDetails.ScheduledEndTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
				),
			)
		}

		sb.WriteString("\n")
	}

	// Livestream has ended
	if video.WasLive() {
		sb.WriteString(
			fmt.Sprintf(
				"🔴 <i>War live von %s bis %s</i>\n",
				video.LiveStreamingDetails.ActualStartTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
				video.LiveStreamingDetails.ActualEndTime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)
	}

	// Blocked
	if video.BlockedInGermany() {
		sb.WriteString("<i>❌ Nicht verfügbar in 🇩🇪</i>\n")
	}

	// Duration
	d, err := video.ContentDetails.ParseDuration()

	if err != nil {
		log.Error().
			Err(err).
			Str("videoID", video.ID).
			Str("duration", video.ContentDetails.Duration).
			Msg("error parsing youtube video duration")
		sb.WriteString(
			fmt.Sprintf(
				"🕒 %s",
				video.ContentDetails.Duration,
			),
		)
	} else {
		if video.IsLive() && !video.IsPremiere() && !video.WasLive() {
			sb.WriteString("🕒 <i>Livestream</i>")
		} else {
			sb.WriteString(
				fmt.Sprintf(
					"🕒 %s",
					utils.HumanizeDuration(d),
				),
			)
		}
	}

	// View count
	if video.IsLiveNow() && video.LiveStreamingDetails.ConcurrentViewers > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 👀 Zurzeit: %s",
				utils.FormatThousand(video.LiveStreamingDetails.ConcurrentViewers),
			),
		)
	}

	if video.Statistics.ViewCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 👁 %s",
				utils.FormatThousand(video.Statistics.ViewCount),
			),
		)
	}

	// Likes
	if video.Statistics.LikeCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 👍 %s",
				utils.FormatThousand(video.Statistics.LikeCount),
			),
		)
	}

	// Comments
	if video.Statistics.CommentCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 💬 %s",
				utils.FormatThousand(video.Statistics.CommentCount),
			),
		)
	}

	return sb.String()
}

func (p *Plugin) OnYouTubeLink(c plugin.GobotContext) error {
	videoID := c.Matches[1]
	video, err := p.getVideoInfo(videoID)

	if err != nil {
		if errors.Is(err, ErrNoVideoFound) {
			return c.Reply("❌ Video nicht gefunden")
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("videoID", videoID).
			Msg("Error while getting video info")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	text := constructText(&video)

	return c.Reply(text, utils.DefaultSendOptions)
}

func (p *Plugin) onYouTubeSearch(c plugin.GobotContext) error {
	query := c.Matches[1]
	_ = c.Notify(telebot.Typing)
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/youtube/v3/search",
	}

	q := requestUrl.Query()
	q.Set("key", p.apiKey)
	q.Set("q", query)
	q.Set("part", "snippet")
	q.Set("maxResults", "1")
	q.Set("type", "video")
	q.Set("fields", "items/id/videoId")

	requestUrl.RawQuery = q.Encode()

	var response SearchResponse
	err := utils.GetRequest(requestUrl.String(), &response)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("error getting youtube search results")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if len(response.Items) == 0 {
		return c.Reply("❌ Keine Ergebnisse gefunden.", utils.DefaultSendOptions)
	}

	videoID := response.Items[0].ID.VideoID
	video, err := p.getVideoInfo(videoID)

	if err != nil {
		if errors.Is(err, ErrNoVideoFound) {
			return c.Reply("❌ Video nicht gefunden")
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("videoID", videoID).
			Msg("Error while getting video info")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("https://www.youtube.com/watch?v=%s\n", video.ID))
	sb.WriteString(constructText(&video))

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:   true,
		DisableNotification: true,
		ParseMode:           telebot.ModeHTML,
	})
}
