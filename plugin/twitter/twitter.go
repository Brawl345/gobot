package twitter

import (
	"encoding/json"
	"fmt"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var log = logger.New("twitter")

type (
	Plugin struct{}
)

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "twitter"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)twitter\.com/\w+/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)twitter\.com/i/web/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)twitter\.com/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)nitter\.net/\w+/status(?:es)?/(\d+)`),
			HandlerFunc: p.OnStatus,
		},
	}
}

func (p *Plugin) OnStatus(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)

	// Get Guest Token first
	var tokenResponse TokenResponse
	req, err := http.NewRequest("POST", activateUrl, nil)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	req.Header.Set("Authorization", bearerToken)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	log.Debug().
		Str("url", activateUrl).
		Interface("headers", req.Header).
		Send()

	resp, err := client.Do(req)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to read guest token body")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if resp.StatusCode != 200 {
		guid := xid.New().String()
		log.Error().
			Str("url", activateUrl).
			Int("status", resp.StatusCode).
			Interface("response", body).
			Msg("Got Twitter HTTP error")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get guest token")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	log.Debug().
		Str("url", activateUrl).
		Interface("response", tokenResponse).
		Send()

	guestToken := tokenResponse.GuestToken

	// Now we get the tweet
	_ = c.Notify(telebot.Typing)
	tweetID := c.Matches[1]
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "api.twitter.com",
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

	requestUrl.RawQuery = q.Encode()

	var tweetResponse TweetResponse
	err = httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"Authorization":             bearerToken,
			"User-Agent":                utils.UserAgent,
			"X-Guest-Token":             guestToken,
			"X-Twitter-Active-User":     "yes",
			"X-Twitter-Client-Language": "de",
			"Authority":                 "api.twitter.com",
		},
		&tweetResponse,
	)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("tweetID", tweetID).
			Msg("Failed to get tweet")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	result := tweetResponse.Tweet(tweetID)
	if result.Typename != "Tweet" && result.Typename != "TweetWithVisibilityResults" {
		if result.Typename == "TweetTombstone" {
			tombstoneText := result.Tombstone.Text.Text
			var sb strings.Builder
			for _, entity := range result.Tombstone.Text.Entities {
				sb.WriteString(tombstoneText[:entity.FromIndex])
				sb.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, entity.Ref.Url, tombstoneText[entity.FromIndex:entity.ToIndex+1]))
			}

			return c.Reply(fmt.Sprintf("❌ %s", sb.String()), utils.DefaultSendOptions)
		}

		return c.Reply("❌ Dieser Tweet existiert nicht.", utils.DefaultSendOptions)
	}

	sendOptions := &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
	}
	var sb strings.Builder
	timezone := utils.GermanTimezone()

	// Tweet author
	sb.WriteString(fmt.Sprintf("%s\n", result.Core.UserResults.Author()))

	// Text
	if result.Legacy.FullText != "" {
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
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
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
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
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
		sb.WriteString(fmt.Sprintf("\n\n<b>⚠️ Leser haben <a href=\"%s\">Kontext</a> hinzugefügt, der ihrer Meinung nach für andere wissenswert wäre:</b>\n", result.BirdwatchPivot.DestinationUrl))
		sb.WriteString(utils.Escape(result.BirdwatchPivot.Note.DataV1.Summary.Text))
	}

	// Quote
	quoteResult := result.QuotedStatusResult.Result
	if quoteResult.Typename == "TweetTombstone" {
		sb.WriteString("\n\n<b>Zitat:</b>\n")

		tombstoneText := quoteResult.Tombstone.Text.Text
		for _, entity := range quoteResult.Tombstone.Text.Entities {
			sb.WriteString(tombstoneText[:entity.FromIndex])
			sb.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, entity.Ref.Url, tombstoneText[entity.FromIndex:entity.ToIndex+1]))
		}
	}

	if quoteResult.Typename == "Tweet" || quoteResult.Typename == "TweetWithVisibilityResults" {
		sb.WriteString("\n\n")

		// Quote author
		sb.WriteString(
			fmt.Sprintf(
				"<b>Zitat von</b> %s\n",
				quoteResult.Core.UserResults.Author(),
			),
		)

		// Quote Text
		if quoteResult.Legacy.FullText != "" {
			// TODO: Withheld
			tweet := quoteResult.Legacy.FullText

			for _, entity := range quoteResult.Legacy.Entities.Urls {
				tweet = strings.ReplaceAll(tweet, entity.Url, entity.ExpandedUrl)
			}

			// Above loop doesn't include e.g. GIFs
			for _, extendedEntity := range quoteResult.Legacy.ExtendedEntities.Media {
				tweet = strings.ReplaceAll(tweet, extendedEntity.Url, extendedEntity.ExpandedUrl)
			}

			sb.WriteString(fmt.Sprintf("%s\n", utils.Escape(tweet)))
		}

		// Quote Poll
		if quoteResult.Card.HasPoll() {
			quotePoll, err := quoteResult.Card.Poll()
			if err != nil {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("tweetID", tweetID).
					Msg("Failed to parse quote poll")
				return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
			}

			sb.WriteString(pollText(quotePoll))
		}

		//	Quote Created + Metrics (RT, Quotes, Likes)
		createdAt, err := time.Parse(time.RubyDate, quoteResult.Legacy.CreatedAt)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("tweetID", tweetID).
				Str("createdAt", quoteResult.Legacy.CreatedAt).
				Msg("Failed to parse quote tweet created at")
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		}
		sb.WriteString(
			fmt.Sprintf(
				"📅 %s",
				createdAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)
		sb.WriteString(quoteResult.Legacy.Metrics())

		// Community Notes / "Birdwatch"
		if quoteResult.BirdwatchPivot.DestinationUrl != "" {
			sb.WriteString(fmt.Sprintf("\n\n<b>⚠️ Leser haben <a href=\"%s\">Kontext</a> hinzugefügt, der ihrer Meinung nach für andere wissenswert wäre:</b>\n", quoteResult.BirdwatchPivot.DestinationUrl))
			sb.WriteString(utils.Escape(quoteResult.BirdwatchPivot.Note.DataV1.Summary.Text))
		}
	}

	// Media
	media := result.Legacy.ExtendedEntities.Media
	if len(media) == 1 && (media[0].IsPhoto() || media[0].IsGIF()) { // One picture or GIF = send as preview
		sendOptions.DisableWebPagePreview = false
		return c.Reply(
			utils.EmbedImage(media[0].Link())+sb.String(),
			sendOptions,
		)
	}

	err = c.Reply(sb.String(), sendOptions)
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
		_ = c.Notify(telebot.UploadingPhoto)
		album := make([]telebot.Inputtable, 0, len(media))

		for _, medium := range media {
			if medium.IsPhoto() {
				album = append(album, &telebot.Photo{Caption: medium.Caption(), File: telebot.FromURL(medium.Link())})
			} else if medium.IsVideo() {
				album = append(album, &telebot.Video{
					Caption: medium.Caption(),
					File:    telebot.FromURL(medium.Link()),
				})
			}
		}

		err := c.SendAlbum(album, telebot.Silent)
		if err != nil {
			// Group send failed - sending media manually as seperate messages
			log.Err(err).Msg("Error while sending album")
			msg, err := c.Bot().Reply(c.Message(),
				"<i>🕒 Medien werden heruntergeladen und gesendet...</i>",
				utils.DefaultSendOptions,
			)
			if err != nil {
				// This would be very awkward
				log.Err(err).Msg("Could not send initial 'download' message")
			}

			for _, medium := range media {
				if medium.IsPhoto() {
					_ = c.Notify(telebot.UploadingPhoto)
				} else {
					_ = c.Notify(telebot.UploadingVideo)
				}

				func() {
					resp, err := httpUtils.HttpClient.Get(medium.Link())
					log.Info().Str("url", medium.Link()).Msg("Downloading")
					if err != nil {
						log.Err(err).Str("url", medium.Link()).Msg("Error while downloading")
						err := c.Reply(medium.Caption(), telebot.Silent, telebot.AllowWithoutReply)
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
						err = c.Reply(&telebot.Photo{File: telebot.FromReader(resp.Body)},
							telebot.Silent, telebot.AllowWithoutReply)
					} else {
						err = c.Reply(&telebot.Video{
							Caption:   medium.Caption(),
							File:      telebot.FromReader(resp.Body),
							Streaming: true,
						}, telebot.Silent, telebot.AllowWithoutReply)
					}
					if err != nil {
						// Last resort: Send URL as text
						log.Err(err).Str("url", medium.Link()).Msg("Error while replying with downloaded medium")
						err := c.Reply(medium.Caption(), telebot.Silent, telebot.AllowWithoutReply)
						if err != nil {
							log.Err(err).Str("url", medium.Link()).Msg("Error while sending medium link")
						}
					}
				}()
			}

			_ = c.Bot().Delete(msg)
		}
	}

	// Now to GIFs...
	if len(gifs) > 0 {
		_ = c.Notify(telebot.UploadingVideo)
		for _, gif := range gifs {

			err = c.Reply(&telebot.Animation{
				Caption: gif.Caption(),
				File:    telebot.FromURL(gif.Link()),
			}, telebot.Silent, telebot.AllowWithoutReply)

			if err != nil {
				func() {
					_ = c.Notify(telebot.UploadingVideo)

					log.Err(err).Str("url", gif.Link()).Msg("Error while sending gif through Telegram")

					resp, err := httpUtils.HttpClient.Get(gif.Link())
					log.Info().Str("url", gif.Link()).Msg("Downloading gif")
					if err != nil {
						log.Err(err).Str("url", gif.Link()).Msg("Error while downloading gif")
						err := c.Reply(gif.Caption(), telebot.Silent, telebot.AllowWithoutReply)
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

					err = c.Reply(&telebot.Animation{
						Caption: gif.Caption(),
						File:    telebot.FromReader(resp.Body),
					}, telebot.Silent, telebot.AllowWithoutReply)

					if err != nil {
						// Last resort: Send URL as text
						log.Err(err).Str("url", gif.Link()).Msg("Error while replying with downloaded gif")
						err := c.Reply(gif.Caption(), telebot.Silent, telebot.AllowWithoutReply)
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
