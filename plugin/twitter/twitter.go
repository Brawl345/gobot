package twitter

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("twitter")

const uploadTimeout = 3 * time.Minute

type (
	Plugin struct {
		mu         sync.Mutex
		guestToken string
	}
)

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "twitter"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:x|twitter)\.com/\w+/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:x|twitter)\.com/i/web/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:x|twitter)\.com/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
	}
}

func (p *Plugin) renewToken() error {
	var tokenResponse TokenResponse
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodPost,
		URL:      activateUrl,
		Headers:  map[string]string{"Authorization": bearerToken},
		Response: &tokenResponse,
	})

	if err != nil {
		return err
	}

	if tokenResponse.GuestToken == "" {
		return errors.New("guest token is empty")
	}

	p.mu.Lock()
	p.guestToken = tokenResponse.GuestToken
	p.mu.Unlock()
	return nil
}

func (p *Plugin) getToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.guestToken
}

func (p *Plugin) fetchTweet(tweetID string) (Result, error) {
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "x.com",
		Path:   tweetDetailsPath,
	}

	q := requestUrl.Query()
	q.Set("variables", fmt.Sprintf(tweetVariables, tweetID))
	q.Set("features", tweetFeatures)
	q.Set("fieldToggles", fieldToggles)
	requestUrl.RawQuery = q.Encode()

	request := func() (Result, error) {
		var tweetResponse TweetResponse
		err := httpUtils.MakeRequest(httpUtils.RequestOptions{
			Method: httpUtils.MethodGet,
			URL:    requestUrl.String(),
			Headers: map[string]string{
				"Authorization":             bearerToken,
				"User-Agent":                utils.UserAgent,
				"X-Guest-Token":             p.getToken(),
				"X-Twitter-Active-User":     "yes",
				"X-Twitter-Client-Language": "de",
			},
			Response: &tweetResponse,
		})
		return tweetResponse.Data.TweetResult.Result, err
	}

	if p.getToken() == "" {
		if err := p.renewToken(); err != nil {
			return Result{}, fmt.Errorf("failed to get guest token: %w", err)
		}
	}

	result, err := request()
	if httpError, ok := errors.AsType[*httpUtils.HttpError](err); ok && httpError.StatusCode == http.StatusForbidden {
		log.Debug().Msg("Renewing guest token")
		if err := p.renewToken(); err != nil {
			return Result{}, fmt.Errorf("failed to renew guest token: %w", err)
		}
		result, err = request()
	}
	return result, err
}

func (p *Plugin) OnStatus(b *gotgbot.Bot, c plugin.GobotContext) error {
	isNsfw := strings.Contains(strings.ToLower(c.EffectiveMessage.GetText()), "nsfw")

	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)
	tweetID := c.Matches[1]

	result, err := p.fetchTweet(tweetID)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("tweetID", tweetID).
			Msg("Failed to get tweet")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if result.Typename == "TweetUnavailable" || result.Typename == "TweetTombstone" {
		if result.Reason == "NsfwLoggedOut" || result.Reason == "NsfwViewerHasNoStatedAge" || result.Tombstone.Typename == "BlurredMediaTombstone" {
			_, err = c.EffectiveMessage.ReplyMessage(b,
				fmt.Sprintf("https://vxtwitter.com/_/status/%s", tweetID),
				&gotgbot.SendMessageOpts{
					ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
					DisableNotification: true,
					ParseMode:           gotgbot.ParseModeHTML,
				},
			)
			return err
		} else if result.Reason == "Protected" {
			_, err := c.EffectiveMessage.ReplyMessage(b, "🔓 Der Account-Inhaber hat beschränkt, wer seine Tweets ansehen kann.", utils.DefaultSendOptions())
			return err
		} else {
			if result.Tombstone.Text.Text != "" {
				tombstoneText := result.Tombstone.Text.Text
				tombstoneText = strings.ReplaceAll(tombstoneText, "Mehr efahren", "")
				_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ %s", utils.Escape(tombstoneText)), utils.DefaultSendOptions())
			} else {
				_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Der Tweet ist nicht einsehbar wegen: <code>%s</code>", utils.Escape(result.Reason)), utils.DefaultSendOptions())
			}
			return err
		}
	}

	if result.Typename != "Tweet" && result.Typename != "TweetWithVisibilityResults" && result.Typename != "tweetResult" {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Dieser Tweet existiert nicht.", utils.DefaultSendOptions())
		return err
	}

	spoiler := isNsfw || result.Legacy.PossiblySensitive
	return sendTweet(b, c, result, spoiler)
}

func sendTweet(b *gotgbot.Bot, c plugin.GobotContext, result Result, spoiler bool) error {
	media := result.Legacy.ExtendedEntities.Media
	prepared := prepareMedia(media)

	msg := renderTweet(result, prepared, mediaPrepared, spoiler)
	err := sendRichMessage(b, c, msg)
	if err == nil {
		return nil
	}
	log.Err(err).Str("tweetID", result.RestId).Int("media", len(msg.media)).Msg("Failed to send rich message")

	if len(prepared) > 0 {
		downloadMissing(prepared)
		msg = renderTweet(result, prepared, mediaUploaded, spoiler)
		err = sendRichMessage(b, c, msg)
		if err == nil {
			return nil
		}
		log.Err(err).Str("tweetID", result.RestId).Int("media", len(msg.media)).Msg("Failed to send rich message with uploaded media")
	}

	msg = renderTweet(result, prepared, mediaNone, spoiler)
	err = sendRichMessage(b, c, msg)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("tweetID", result.RestId).
			Str("html", msg.html.String()).
			Msg("Failed to send rich message without media")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	sendAlbums(b, c, media, spoiler)
	return nil
}

type richMessageUpload struct {
	gotgbot.InputRichMessage
}

func (r richMessageUpload) Attach(_ string, w *multipart.Writer) error {
	for _, m := range r.Media {
		if err := m.Media.Attach("file_"+m.Id, w); err != nil {
			return err
		}
	}
	return nil
}

func sendRichMessage(b *gotgbot.Bot, c plugin.GobotContext, msg *tweetMessage) error {
	var opts *gotgbot.RequestOpts
	if msg.uploads > 0 {
		_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadVideo, nil)
		opts = &gotgbot.RequestOpts{Timeout: uploadTimeout}
	}

	_, err := b.Request("sendRichMessage", map[string]any{
		"chat_id": c.EffectiveChat.Id,
		"rich_message": richMessageUpload{gotgbot.InputRichMessage{
			Html:                msg.html.String(),
			Media:               msg.media,
			SkipEntityDetection: true,
		}},
		"reply_parameters": gotgbot.ReplyParameters{
			MessageId:                c.EffectiveMessage.MessageId,
			AllowSendingWithoutReply: true,
		},
		"disable_notification": true,
	}, opts)
	return err
}

func pollText(poll Poll) string {
	timezone := utils.GermanTimezone()

	var sb strings.Builder

	sb.WriteString("\n<i>📊 Umfrage:")
	if poll.Closed() {
		sb.WriteString(" (beendet)")
	}
	sb.WriteString("</i>\n")

	for _, option := range poll.Options {
		plural := ""
		if option.Votes != 1 {
			plural = "n"
		}
		percentage := (float64(option.Votes) / float64(poll.TotalVotes)) * 100
		sb.WriteString(
			fmt.Sprintf(
				"%d) %s <i>(%s Stimme%s, %.1f %%)</i>\n",
				option.Position,
				utils.Escape(option.Label),
				utils.FormatThousand(option.Votes),
				plural,
				percentage,
			),
		)
	}

	var plural string
	if poll.TotalVotes != 1 {
		plural = "n"
	}

	var closed string
	if poll.Closed() {
		closed = "e"
	}

	sb.WriteString(
		fmt.Sprintf(
			"\n<i>%s Stimme%s - endet%s am %s</i>\n\n",
			utils.FormatThousand(poll.TotalVotes),
			plural,
			closed,
			poll.EndDatetime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
		),
	)

	return sb.String()
}
