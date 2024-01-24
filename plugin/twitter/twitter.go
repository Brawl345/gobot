package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("twitter")

const (
	MaxNoteLength = 500
)

type (
	Plugin struct{}
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
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)nitter\.net/\w+/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
	}
}

func (p *Plugin) OnStatus(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)

	// Get Guest Token first
	var tokenResponse TokenResponse
	req, err := http.NewRequest("POST", activateUrl, nil)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	req.Header.Set("Authorization", bearerToken)

	log.Debug().
		Str("url", activateUrl).
		Interface("headers", req.Header).
		Send()

	resp, err := httpUtils.HttpClient.Do(req)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("error closing body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to read guest token body")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if resp.StatusCode != 200 {
		guid := xid.New().String()
		log.Error().
			Str("url", activateUrl).
			Int("status", resp.StatusCode).
			Interface("response", body).
			Msg("Got Twitter HTTP error")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	log.Debug().
		Str("url", activateUrl).
		Interface("response", tokenResponse).
		Send()

	guestToken := tokenResponse.GuestToken

	// Now we get the tweet
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	tweetID := c.Matches[1]
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "twitter.com",
		Path:   tweetDetailsPath,
	}

	q := requestUrl.Query()

	q.Set(
		"variables",
		fmt.Sprintf(tweetVariables, tweetID),
	)

	q.Set(
		"features",
		tweetFeatures,
	)

	q.Set(
		"fieldToggles",
		fieldToggles,
	)

	requestUrl.RawQuery = q.Encode()

	var tweetResponse TweetResponse
	err = httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"Authorization":             bearerToken,
			"User-Agent":                "Googlebot",
			"X-Guest-Token":             guestToken,
			"X-Twitter-Active-User":     "yes",
			"X-Twitter-Client-Language": "de",
		},
		&tweetResponse,
	)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("tweetID", tweetID).
			Msg("Failed to get tweet")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	result := tweetResponse.Data.TweetResult.Result

	if result.Typename == "TweetUnavailable" {
		if result.Reason == "NsfwLoggedOut" {
			_, err = c.EffectiveMessage.Reply(b,
				fmt.Sprintf("https://vxtwitter.com/_/status/%s", tweetID),
				&gotgbot.SendMessageOpts{
					ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
					DisableNotification: true,
					ParseMode:           gotgbot.ParseModeHTML,
				},
			)
			return err
		} else if result.Reason == "Protected" {
			_, err := c.EffectiveMessage.Reply(b, "🔓 Der Account-Inhaber hat beschränkt, wer seine Tweets ansehen kann.", utils.DefaultSendOptions())
			return err
		} else {
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Der Tweet ist nicht einsehbar wegen: <code>%s</code>", result.Reason), utils.DefaultSendOptions())
			return err
		}
	}

	if result.Typename != "Tweet" && result.Typename != "TweetWithVisibilityResults" && result.Typename != "tweetResult" {
		_, err := c.EffectiveMessage.Reply(b, "❌ Dieser Tweet existiert nicht.", utils.DefaultSendOptions())
		return err
	}

	sendOptions := &gotgbot.SendMessageOpts{
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	}
	var sb strings.Builder
	timezone := utils.GermanTimezone()

	// Tweet author
	sb.WriteString(fmt.Sprintf("%s\n", result.Core.UserResults.Author()))

	// Text
	if result.NoteTweet.NoteTweetResults.Result.Text != "" {
		tweet := result.NoteTweet.NoteTweetResults.Result.Text

		for _, entity := range result.NoteTweet.NoteTweetResults.Result.EntitySet.Urls {
			tweet = strings.ReplaceAll(tweet, entity.Url, entity.ExpandedUrl)
		}

		if len(tweet) > MaxNoteLength {
			tweet = fmt.Sprintf("%s...\n<a href=\"https://twitter.com/%s/status/%s\">Weiterlesen...</a>",
				utils.Escape(tweet[:MaxNoteLength]),
				result.Core.UserResults.Result.Legacy.ScreenName,
				result.RestId,
			)
		}

		sb.WriteString(fmt.Sprintf("%s\n", tweet))
	} else if result.Legacy.FullText != "" {
		// TODO: Withheld
		tweet := result.Legacy.FullText

		for _, entity := range result.Legacy.Entities.Urls {
			tweet = strings.ReplaceAll(tweet, entity.Url, entity.ExpandedUrl)
		}

		// Above loop doesn't include e.g. GIFs
		for _, extendedEntity := range result.Legacy.ExtendedEntities.Media {
			tweet = strings.ReplaceAll(tweet, extendedEntity.Url, "")
		}

		sb.WriteString(fmt.Sprintf("%s\n", utils.Escape(tweet)))
	}

	// Poll
	if result.Card.HasPoll() {
		poll, err := result.Card.Poll()
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("tweetID", tweetID).
				Msg("Failed to parse poll")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		sb.WriteString(pollText(poll))
	}

	//	Created + Metrics (RT, Quotes, Likes, Bookmarks)
	createdAt, err := time.Parse(time.RubyDate, result.Legacy.CreatedAt)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("tweetID", tweetID).
			Str("createdAt", result.Legacy.CreatedAt).
			Msg("Failed to parse tweet created at")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}
	sb.WriteString(
		fmt.Sprintf(
			"📅 %s",
			createdAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
		),
	)
	sb.WriteString(result.Legacy.Metrics())

	// Community Notes / "Birdwatch"
	if result.BirdwatchPivot.DestinationUrl != "" {
		sb.WriteString(fmt.Sprintf("\n\n<b>⚠️ Leser haben <a href=\"%s\">Kontext</a> hinzugefügt, der ihrer Meinung nach für andere wissenswert wäre.</b>", result.BirdwatchPivot.DestinationUrl))

		// TODO: Links need to be replaced and it's kinda annoying
		//sb.WriteString(utils.Escape(result.BirdwatchPivot.Subtitle.Text))
	}

	// Quote
	quoteResult := result.QuotedStatusResult.Result

	if quoteResult.Typename == "TweetUnavailable" {
		if quoteResult.Reason == "NsfwLoggedOut" {
			sb.WriteString("<i>Tweet kann nicht angezeigt werden, weil er sensible Inhalte enthält.</i>")
		} else if quoteResult.Reason == "Protected" {
			sb.WriteString("\"<i>🔓 Der Account-Inhaber hat beschränkt, wer seine Tweets ansehen kann.</i>")
		} else {
			sb.WriteString(fmt.Sprintf("<i>❌ Der Tweet ist nicht einsehbar wegen: <code>%s</code></i>", result.Reason))
		}
	}

	if quoteResult.Typename == "Tweet" || quoteResult.Typename == "TweetWithVisibilityResults" {
		sb.WriteString("\n\n")

		quoteResultSub := quoteResult.Tweet
		if quoteResult.TweetSub.RestId != "" {
			quoteResultSub = quoteResult.TweetSub
		}

		// Quote author
		sb.WriteString(
			fmt.Sprintf(
				"<b>Zitat von</b> %s\n",
				quoteResultSub.Core.UserResults.Author(),
			),
		)

		// Quote Text
		if quoteResultSub.NoteTweet.NoteTweetResults.Result.Text != "" {
			tweet := quoteResultSub.NoteTweet.NoteTweetResults.Result.Text

			for _, entity := range quoteResultSub.NoteTweet.NoteTweetResults.Result.EntitySet.Urls {
				tweet = strings.ReplaceAll(tweet, entity.Url, entity.ExpandedUrl)
			}

			if len(tweet) > MaxNoteLength {
				tweet = fmt.Sprintf("%s...\n<a href=\"https://twitter.com/%s/status/%s\">Zitat Weiterlesen...</a>",
					utils.Escape(tweet[:MaxNoteLength]),
					quoteResultSub.Core.UserResults.Result.Legacy.ScreenName,
					quoteResultSub.RestId,
				)
			}

			sb.WriteString(fmt.Sprintf("%s\n", tweet))
		} else if quoteResultSub.Legacy.FullText != "" {
			// TODO: Withheld
			tweet := quoteResultSub.Legacy.FullText

			for _, entity := range quoteResultSub.Legacy.Entities.Urls {
				tweet = strings.ReplaceAll(tweet, entity.Url, entity.ExpandedUrl)
			}

			// Above loop doesn't include e.g. GIFs
			for _, extendedEntity := range quoteResultSub.Legacy.ExtendedEntities.Media {
				tweet = strings.ReplaceAll(tweet, extendedEntity.Url, extendedEntity.ExpandedUrl)
			}

			sb.WriteString(fmt.Sprintf("%s\n", utils.Escape(tweet)))
		}

		// Quote Poll
		if quoteResultSub.Card.HasPoll() {
			quotePoll, err := quoteResultSub.Card.Poll()
			if err != nil {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("tweetID", tweetID).
					Msg("Failed to parse quote poll")
				_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
				return err
			}

			sb.WriteString(pollText(quotePoll))
		}

		//	Quote Created + Metrics (RT, Quotes, Likes)
		createdAt, err := time.Parse(time.RubyDate, quoteResultSub.Legacy.CreatedAt)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("tweetID", tweetID).
				Str("createdAt", quoteResultSub.Legacy.CreatedAt).
				Msg("Failed to parse quote tweet created at")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}
		sb.WriteString(
			fmt.Sprintf(
				"📅 %s",
				createdAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)
		sb.WriteString(quoteResultSub.Legacy.Metrics())

		// Community Notes / "Birdwatch"
		if quoteResultSub.BirdwatchPivot.DestinationUrl != "" {
			sb.WriteString(fmt.Sprintf("\n\n<b>⚠️ Leser haben <a href=\"%s\">Kontext</a> hinzugefügt, der ihrer Meinung nach für andere wissenswert wäre.</b>", quoteResultSub.BirdwatchPivot.DestinationUrl))

			// TODO: Links need to be replaced and it's kinda annoying
			//sb.WriteString(utils.Escape(quoteResultSub.BirdwatchPivot.Subtitle.Text))
		}
	}

	// Media
	media := result.Legacy.ExtendedEntities.Media
	if len(media) == 1 && (media[0].IsPhoto() || media[0].IsGIF()) { // One picture or GIF = send as preview
		sendOptions.LinkPreviewOptions.IsDisabled = false
		sendOptions.LinkPreviewOptions.Url = media[0].Link()
		sendOptions.LinkPreviewOptions.PreferLargeMedia = true
		_, err := c.EffectiveMessage.Reply(b,
			sb.String(),
			sendOptions,
		)
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, sb.String(), sendOptions)
	if err != nil {
		return err
	}

	// Multiple media = send all as album
	// NOTE: Telegram does not support sending multiple *animations/GIFs* in an album
	//	so we will handle them seperately
	gifs := make([]Medium, 0, len(media))
	for _, medium := range media {
		if medium.IsGIF() {
			gifs = append(gifs, medium)
		}
	}

	if len(media) > 0 && len(media) != len(gifs) {
		// Try album (photos + videos, no GIFs) first
		_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)
		album := make([]gotgbot.InputMedia, 0, len(media))

		for _, medium := range media {
			if medium.IsPhoto() {
				album = append(album, gotgbot.InputMediaPhoto{Caption: medium.Caption(), Media: medium.Link()})
			} else if medium.IsVideo() {
				album = append(album, gotgbot.InputMediaVideo{Caption: medium.Caption(), Media: medium.Link()})
			}
		}

		_, err := b.SendMediaGroup(
			c.EffectiveChat.Id,
			album,
			&gotgbot.SendMediaGroupOpts{DisableNotification: true,
				ReplyParameters: &gotgbot.ReplyParameters{
					MessageId: c.EffectiveMessage.MessageId,
				},
			},
		)
		if err != nil {
			// Group send failed - sending media manually as seperate messages
			log.Err(err).Msg("Error while sending album")
			msg, err := c.EffectiveMessage.Reply(b,
				"<i>🕒 Medien werden heruntergeladen und gesendet...</i>",
				utils.DefaultSendOptions(),
			)
			if err != nil {
				// This would be very awkward
				log.Err(err).Msg("Could not send initial 'download' message")
			}

			for _, medium := range media {
				if medium.IsPhoto() {
					_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)
				} else {
					_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadVideo, nil)
				}

				func() {
					resp, err := httpUtils.HttpClient.Get(medium.Link())
					log.Info().Str("url", medium.Link()).Msg("Downloading")
					if err != nil {
						log.Err(err).Str("url", medium.Link()).Msg("Error while downloading")
						_, err := c.EffectiveMessage.Reply(b, medium.Caption(), &gotgbot.SendMessageOpts{
							ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
							DisableNotification: true,
						})
						if err != nil {
							log.Err(err).Str("url", medium.Link()).Msg("Error while replying with link")
						}
						return
					}

					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							log.Err(err).Msg("Error while closing body")
						}
					}(resp.Body)

					if medium.IsPhoto() {
						_, err = b.SendPhoto(c.EffectiveChat.Id, resp.Body, &gotgbot.SendPhotoOpts{
							ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true,
								MessageId: c.EffectiveMessage.MessageId},
							DisableNotification: true,
						})
					} else {
						_, err = b.SendVideo(c.EffectiveChat.Id, resp.Body, &gotgbot.SendVideoOpts{
							Caption: medium.Caption(),
							ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true,
								MessageId: c.EffectiveMessage.MessageId},
							DisableNotification: true,
							SupportsStreaming:   true,
						})
					}
					if err != nil {
						// Last resort: Send URL as text
						log.Err(err).Str("url", medium.Link()).Msg("Error while replying with downloaded medium")
						_, err := c.EffectiveMessage.Reply(b, medium.Caption(), &gotgbot.SendMessageOpts{
							ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
							DisableNotification: true,
						})
						if err != nil {
							log.Err(err).Str("url", medium.Link()).Msg("Error while sending medium link")
						}
					}
				}()
			}

			_, _ = msg.Delete(b, nil)
		}
	}

	// Now to GIFs...
	if len(gifs) > 0 {
		_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadVideo, nil)
		for _, gif := range gifs {

			_, err := b.SendAnimation(c.EffectiveChat.Id,
				gif.Link(),
				&gotgbot.SendAnimationOpts{
					Caption: gif.Caption(),
					ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true,
						MessageId: c.EffectiveMessage.MessageId},
					DisableNotification: true,
				},
			)

			if err != nil {
				func() {
					_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadVideo, nil)

					log.Err(err).Str("url", gif.Link()).Msg("Error while sending gif through Telegram")

					resp, err := httpUtils.HttpClient.Get(gif.Link())
					log.Info().Str("url", gif.Link()).Msg("Downloading gif")
					if err != nil {
						log.Err(err).Str("url", gif.Link()).Msg("Error while downloading gif")
						_, err := c.EffectiveMessage.Reply(b, gif.Caption(), &gotgbot.SendMessageOpts{
							ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
							DisableNotification: true,
						})
						if err != nil {
							log.Err(err).Str("url", gif.Link()).Msg("Error while replying with link")
						}
						return
					}

					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							log.Err(err).Msg("Error while closing body")
						}
					}(resp.Body)

					_, err = b.SendAnimation(c.EffectiveChat.Id, resp.Body, &gotgbot.SendAnimationOpts{
						Caption: gif.Caption(),
						ReplyParameters: &gotgbot.ReplyParameters{AllowSendingWithoutReply: true,
							MessageId: c.EffectiveMessage.MessageId},
						DisableNotification: true,
					})

					if err != nil {
						// Last resort: Send URL as text
						log.Err(err).Str("url", gif.Link()).Msg("Error while replying with downloaded gif")
						_, err := c.EffectiveMessage.Reply(b, gif.Caption(), &gotgbot.SendMessageOpts{
							ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
							DisableNotification: true,
						})
						if err != nil {
							log.Err(err).Str("url", gif.Link()).Msg("Error while sending gif link")
						}
					}
				}()
			}
		}

	}

	return nil
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
