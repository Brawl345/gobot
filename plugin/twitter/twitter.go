package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"gopkg.in/telebot.v3"
)

var log = logger.New("twitter")

type (
	Plugin struct {
		bearerToken string
	}
)

func New(credentialService model.CredentialService) *Plugin {
	bearerToken, err := credentialService.GetKey("twitter_bearer_token")
	if err != nil {
		log.Warn().Msg("twitter_bearer_token not found")
	}

	return &Plugin{
		bearerToken: bearerToken,
	}
}

func doTwitterRequest(url string, bearerToken string, result *Response) error {
	log.Debug().
		Str("url", url).
		Send()
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing body")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var twitterError Error
	if resp.StatusCode != 200 {
		if err := json.Unmarshal(body, &twitterError); err != nil {
			return &httpUtils.HttpError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
			}
		}
		log.Error().
			Str("url", url).
			Interface("response", twitterError).
			Msg("Got Twitter error")
		return &twitterError
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	var partialError PartialError
	err = json.Unmarshal(body, &partialError)
	if err == nil && partialError.Errors != nil {
		for _, pe := range partialError.Errors {
			// Ignore partial errors for tweets that are not the requested one
			if pe.ResourceId == result.Tweet.ID || result.Tweet.ID == "" {
				log.Error().
					Str("url", url).
					Interface("response", partialError).
					Msg("Got partial Twitter error")
				return &partialError
			}
		}
	}

	log.Debug().
		Str("url", url).
		Interface("response", result).
		Send()

	return nil
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
	var httpError *httpUtils.HttpError
	var partialError *PartialError
	var twitterError *Error
	var response Response

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "api.twitter.com",
		Path:   fmt.Sprintf("/2/tweets/%s", c.Matches[1]),
	}

	q := requestUrl.Query()

	// https://developer.twitter.com/en/docs/twitter-api/expansions#:~:text=Available%20expansions%20in%20a%20Tweet%20payload
	q.Set(
		"expansions",
		"attachments.media_keys,attachments.poll_ids,author_id,referenced_tweets.id,referenced_tweets.id.author_id",
	)

	// https://developer.twitter.com/en/docs/twitter-api/data-dictionary/object-model/media
	q.Set(
		"media.fields",
		"alt_text,media_key,public_metrics,type,url,variants",
	)

	// https://developer.twitter.com/en/docs/twitter-api/data-dictionary/object-model/poll
	q.Set(
		"poll.fields",
		"id,options,end_datetime,voting_status",
	)

	// https://developer.twitter.com/en/docs/twitter-api/data-dictionary/object-model/tweet
	q.Set(
		"tweet.fields",
		"created_at,entities,public_metrics,withheld",
	)

	// https://developer.twitter.com/en/docs/twitter-api/data-dictionary/object-model/user
	q.Set(
		"user.fields",
		"id,username,name,verified,protected",
	)

	requestUrl.RawQuery = q.Encode()

	err := doTwitterRequest(
		requestUrl.String(),
		p.bearerToken,
		&response,
	)

	if err != nil {
		if errors.As(err, &httpError) {
			log.Error().Int("status_code", httpError.StatusCode).Msg("Unexpected status code")
		} else if errors.As(err, &twitterError) { // Log only errors that are not "status not found"
			for _, err := range twitterError.Errors {
				for param := range err.Parameters {
					if param == "id" {
						return c.Reply("❌ Der Status wurde nicht gefunden.")
					}
				}
			}
			log.Err(twitterError).Interface("error", twitterError.Errors).Send()
		} else if errors.As(err, &partialError) {
			for _, err := range partialError.Errors {
				if err.Title == "Not Found Error" {
					return c.Reply("❌ Der Status wurde nicht gefunden.")
				}
				if err.Title == "Authorization Error" {
					return c.Reply("❌ Die Tweets dieses Nutzers sind privat.")
				}
			}
			return c.Reply(fmt.Sprintf("❌ <b>API-Fehler:</b> %s", utils.Escape(partialError.Errors[0].Detail)),
				utils.DefaultSendOptions)
		} else {
			log.Err(err).Send()
		}
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
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
	author := response.Includes.User(response.Tweet.AuthorID)
	sb.WriteString(fmt.Sprintf("%s\n", author.String()))

	// Text
	if response.Tweet.Text != "" && !(response.Tweet.Withheld.InGermany() && response.Tweet.Withheld.Scope == "tweet") {
		tweet := response.Tweet.Text
		for _, entityURL := range response.Tweet.Entities.URLs {
			if entityURL.MediaKey != "" || strings.Contains(entityURL.ExpandedUrl, response.Tweet.ID) {
				// GIFs don't have a mediaKey so we don't even know if the URL points to a GIF...
				tweet = strings.ReplaceAll(tweet, entityURL.Url, "")
			} else {
				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.Expand())
			}
		}

		if tweet != "" { // Do not insert a blank line when there is only a media attachment without text
			sb.WriteString(
				fmt.Sprintf(
					"%s\n",
					utils.Escape(tweet),
				),
			)
		}
	}

	// Withheld info
	if response.Tweet.Withheld.InGermany() {
		sb.WriteString(fmt.Sprintf("%s\n", response.Tweet.Withheld.String()))
	}

	// Poll
	if len(response.Includes.Polls) > 0 {
		poll := response.Includes.Polls[0]
		totalVotes := poll.TotalVotes()
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
			percentage := (float64(option.Votes) / float64(totalVotes)) * 100
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
		if totalVotes != 1 {
			plural = "n"
		}

		var closed string
		if poll.Closed() {
			closed = "e"
		}

		sb.WriteString(
			fmt.Sprintf(
				"\n<i>%s Stimme%s - endet%s am %s</i>\n\n",
				utils.FormatThousand(totalVotes),
				plural,
				closed,
				poll.EndDatetime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)

	}

	// Created + Metrics (RT, Quotes, Likes)
	sb.WriteString(
		fmt.Sprintf(
			"📅 %s",
			response.Tweet.CreatedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
		),
	)
	sb.WriteString(response.Tweet.PublicMetrics.String())

	// Quote
	quote := response.Quote()
	if quote != nil {
		sb.WriteString("\n\n")

		// Quote author
		quoteAuthor := response.Includes.User(quote.AuthorID)
		sb.WriteString(
			fmt.Sprintf(
				"<b>Zitat von</b> %s\n",
				quoteAuthor.String(),
			),
		)

		// Quote text
		if quote.Text != "" && !(quote.Withheld.InGermany() && quote.Withheld.Scope == "tweet") {
			tweet := quote.Text
			for _, entityURL := range quote.Entities.URLs {
				// Same as for normal tweets, but don't remove media links
				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.Expand())
			}

			sb.WriteString(
				fmt.Sprintf(
					"%s\n",
					utils.Escape(tweet),
				),
			)
		}

		// Quote withheld info
		if quote.Withheld.InGermany() {
			sb.WriteString(fmt.Sprintf("%s\n", quote.Withheld.String()))
		}

		// Quote poll (only link since the object isn't returned)
		if len(quote.Attachments.PollIDs) > 0 {
			sb.WriteString(
				fmt.Sprintf(
					"📊 <i>Dieser Tweet enthält eine Umfrage - <a href=\"https://twitter.com/%s/status/%s\">rufe ihn im Browser auf</a>, um sie anzuzeigen</i>\n",
					quoteAuthor.Username,
					quote.ID,
				),
			)
		}

		// Quote created at + metrics (RT, Quotes, Likes)
		sb.WriteString(
			fmt.Sprintf(
				"📅 %s",
				quote.CreatedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)
		sb.WriteString(quote.PublicMetrics.String())

	}

	// Media
	media := response.Includes.Media

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
	gifs := make([]Media, 0, len(media))
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
