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
	if result.HasPoll() {
		poll, err := result.Poll()
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("tweetID", tweetID).
				Msg("Failed to parse poll")
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		}

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

	}

	// TODO: Community Notes / Birdwatch?

	//	Created + Metrics (RT, Quotes, Likes)
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

//	// Text
//	if response.Tweet.Text != "" && !(response.Tweet.Withheld.InGermany() && response.Tweet.Withheld.Scope == "tweet") {
//		tweet := response.Tweet.Text
//		for _, entityURL := range response.Tweet.Entities.URLs {
//			if entityURL.MediaKey != "" || strings.Contains(entityURL.ExpandedUrl, response.Tweet.ID) {
//				// GIFs don't have a mediaKey so we don't even know if the URL points to a GIF...
//				tweet = strings.ReplaceAll(tweet, entityURL.Url, "")
//			} else {
//				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.Expand())
//			}
//		}
//
//		if tweet != "" { // Do not insert a blank line when there is only a media attachment without text
//			sb.WriteString(
//				fmt.Sprintf(
//					"%s\n",
//					utils.Escape(tweet),
//				),
//			)
//		}
//	}
//
//	// Withheld info
//	if response.Tweet.Withheld.InGermany() {
//		sb.WriteString(fmt.Sprintf("%s\n", response.Tweet.Withheld.String()))
//	}
//
//	// Quote
//	quote := response.Quote()
//	if quote != nil {
//		sb.WriteString("\n\n")
//
//		// Quote author
//		quoteAuthor := response.Includes.User(quote.AuthorID)
//		sb.WriteString(
//			fmt.Sprintf(
//				"<b>Zitat von</b> %s\n",
//				quoteAuthor.String(),
//			),
//		)
//
//		// Quote text
//		if quote.Text != "" && !(quote.Withheld.InGermany() && quote.Withheld.Scope == "tweet") {
//			tweet := quote.Text
//			for _, entityURL := range quote.Entities.URLs {
//				// Same as for normal tweets, but don't remove media links
//				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.Expand())
//			}
//
//			sb.WriteString(
//				fmt.Sprintf(
//					"%s\n",
//					utils.Escape(tweet),
//				),
//			)
//		}
//
//		// Quote withheld info
//		if quote.Withheld.InGermany() {
//			sb.WriteString(fmt.Sprintf("%s\n", quote.Withheld.String()))
//		}
//
//		// Quote poll (only link since the object isn't returned)
//		if len(quote.Attachments.PollIDs) > 0 {
//			sb.WriteString(
//				fmt.Sprintf(
//					"📊 <i>Dieser Tweet enthält eine Umfrage - <a href=\"https://twitter.com/%s/status/%s\">rufe ihn im Browser auf</a>, um sie anzuzeigen</i>\n",
//					quoteAuthor.Username,
//					quote.ID,
//				),
//			)
//		}
//
//		// Quote created at + metrics (RT, Quotes, Likes)
//		sb.WriteString(
//			fmt.Sprintf(
//				"📅 %s",
//				quote.CreatedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
//			),
//		)
//		sb.WriteString(quote.PublicMetrics.String())
//
//	}
//}
