package youtube

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("youtube")

type Plugin struct {
	credentialService model.CredentialService
}

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (p *Plugin) Name() string {
	return "youtube"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "yt",
			Description: "<Suchbegriff> - Auf YouTube suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	// For videoId see https://webapps.stackexchange.com/a/101153
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)youtube\.com/watch(?:\?|\?.+&)?v=([\dA-Za-z_-]{10}[048AEIMQUYcgkosw])`),
			HandlerFunc: p.OnYouTubeLink,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)youtube\.com/(?:embed|shorts|live)/([\dA-Za-z_-]{10}[048AEIMQUYcgkosw])`),
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
	apiKey := p.credentialService.GetKey("google_api_key")
	if apiKey == "" {
		log.Warn().Msg("google_api_key not found")
		return Video{}, errors.New("google_api_key not found")
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/youtube/v3/videos",
	}

	q := requestUrl.Query()
	q.Set("key", apiKey)
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
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Response: &response,
	})

	if err != nil {
		return Video{}, err
	}

	if len(response.Items) == 0 {
		return Video{}, ErrNoVideoFound
	}

	return response.Items[0], nil
}

func deArrow(originalText string, video *Video) (string, error) {
	// https://wiki.sponsor.ajay.app/w/API_Docs/DeArrow#GET_/api/branding
	deArrowUrl := fmt.Sprintf("https://sponsor.ajay.app/api/branding/?videoID=%s", video.ID)
	var deArrowResponse DeArrowResponse
	var httpError *httpUtils.HttpError
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      deArrowUrl,
		Response: &deArrowResponse,
	})

	if err != nil {
		if errors.As(err, &httpError) {
			if httpError.StatusCode == http.StatusInternalServerError ||
				httpError.StatusCode == http.StatusNotFound { // API seems to throw 500 for some empty responses
				return "", nil
			}
		}
		return "", err
	}

	alternativeTitle := deArrowResponse.GetBestTitle()
	if alternativeTitle != "" {
		modifiedText := strings.Replace(
			originalText,
			fmt.Sprintf("<b>%s</b>\n", utils.Escape(video.Snippet.Title)),
			fmt.Sprintf("<b>%s</b>\n<i>Alternativer Titel: <b>%s</b>\n</i>",
				utils.Escape(video.Snippet.Title),
				utils.Escape(alternativeTitle),
			),
			1,
		)
		return modifiedText, nil
	}

	return "", nil
}

func constructText(video *Video) string {
	var sb strings.Builder

	// Title
	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b>\n",
			utils.Escape(video.Snippet.Title),
		),
	)

	// Uploader
	sb.WriteString(
		fmt.Sprintf(
			"🎥 <strong><a href=\"https://www.youtube.com/channel/%s/videos\">%s</a></strong>",
			video.Snippet.ChannelID,
			utils.Escape(video.Snippet.ChannelTitle),
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

func (p *Plugin) OnYouTubeLink(b *gotgbot.Bot, c plugin.GobotContext) error {
	videoID := c.Matches[1]
	video, err := p.getVideoInfo(videoID)

	if err != nil {
		if errors.Is(err, ErrNoVideoFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Video nicht gefunden", nil)
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("videoID", videoID).
			Msg("Error while getting video info")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	text := constructText(&video)

	msg, err := c.EffectiveMessage.Reply(b, text, utils.DefaultSendOptions())
	if err == nil {
		modifiedText, err := deArrow(text, &video)
		if err != nil {
			log.Err(err).
				Str("videoID", videoID).
				Msg("Error while contacting DeArrow API")
			return nil
		}

		_, _, err = msg.EditText(b, modifiedText, &gotgbot.EditMessageTextOpts{
			ParseMode: gotgbot.ParseModeHTML,
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled: true,
			},
		})
	}

	return err
}

func (p *Plugin) onYouTubeSearch(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)

	apiKey := p.credentialService.GetKey("google_api_key")
	if apiKey == "" {
		log.Warn().Msg("google_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>google_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	query := c.Matches[1]
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/youtube/v3/search",
	}

	q := requestUrl.Query()
	q.Set("key", apiKey)
	q.Set("q", query)
	q.Set("part", "snippet")
	q.Set("maxResults", "1")
	q.Set("type", "video")
	q.Set("fields", "items/id/videoId")

	requestUrl.RawQuery = q.Encode()

	var response SearchResponse
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Response: &response,
	})

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("error getting youtube search results")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Items) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Ergebnisse gefunden.", utils.DefaultSendOptions())
		return err
	}

	videoID := response.Items[0].ID.VideoID
	video, err := p.getVideoInfo(videoID)

	if err != nil {
		if errors.Is(err, ErrNoVideoFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Video nicht gefunden", nil)
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("videoID", videoID).
			Msg("Error while getting video info")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("https://www.youtube.com/watch?v=%s\n", video.ID))
	sb.WriteString(constructText(&video))
	text := sb.String()

	msg, err := c.EffectiveMessage.Reply(b, text, &gotgbot.SendMessageOpts{
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	})

	if err == nil {
		modifiedText, err := deArrow(text, &video)
		if err != nil {
			log.Err(err).
				Str("videoID", videoID).
				Msg("Error while contacting DeArrow API")
			return nil
		}

		_, _, err = msg.EditText(b, modifiedText, &gotgbot.EditMessageTextOpts{
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	return err
}
