package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.New("twitter")

type (
	Plugin struct {
		bearerToken    string
		consumerKey    string
		consumerSecret string
		accessToken    string
		accessSecret   string
	}
)

func New(credentialService models.CredentialService) *Plugin {
	bearerToken, err := credentialService.GetKey("twitter_bearer_token")
	if err != nil {
		log.Warn().Msg("twitter_bearer_token not found")
	}

	consumerKey, err := credentialService.GetKey("twitter_consumer_key")
	if err != nil {
		log.Warn().Msg("twitter_consumer_key not found")
	}

	consumerSecret, err := credentialService.GetKey("twitter_consumer_secret")
	if err != nil {
		log.Warn().Msg("twitter_consumer_secret not found")
	}

	accessToken, err := credentialService.GetKey("twitter_access_token_key")
	if err != nil {
		log.Warn().Msg("twitter_access_token_key not found")
	}

	accessSecret, err := credentialService.GetKey("twitter_access_token_secret")
	if err != nil {
		log.Warn().Msg("twitter_access_token_secret not found")
	}

	return &Plugin{
		bearerToken:    bearerToken,
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		accessToken:    accessToken,
		accessSecret:   accessSecret,
	}
}

func doTwitterRequest(url string, bearerToken string, result any) error {
	log.Debug().
		Str("url", url).
		Send()
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	client := &http.Client{}
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
			return &utils.HttpError{
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

	var partialError PartialError

	err = json.Unmarshal(body, &partialError)
	if err == nil && partialError.Errors != nil {
		log.Error().
			Str("url", url).
			Interface("response", partialError).
			Msg("Got partial Twitter error")
		return &partialError
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
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

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("(?i)twitter\\.com/\\w+/status(?:es)?/(\\d+)"),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("(?i)twitter\\.com/status(?:es)?/(\\d+)"),
			HandlerFunc: p.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("(?i)nitter\\.net/\\w+/status(?:es)?/(\\d+)"),
			HandlerFunc: p.OnStatus,
		},
	}
}

func (p *Plugin) OnStatus(c plugin.GobotContext) error {
	var httpError *utils.HttpError
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
		"alt_text,media_key,public_metrics,type,url",
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
			return c.Reply(fmt.Sprintf("❌ <b>API-Fehler:</b> %s", html.EscapeString(partialError.Errors[0].Detail)),
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
			if strings.Contains(entityURL.ExpandedUrl, response.Tweet.ID) {
				tweet = strings.ReplaceAll(tweet, entityURL.Url, "")
			} else {
				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.ExpandedUrl)
			}
		}

		if tweet != "" { // Do not insert a blank line when there is only a media attachment without text
			sb.WriteString(
				fmt.Sprintf(
					"%s\n",
					html.EscapeString(tweet),
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
					html.EscapeString(option.Label),
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
				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.ExpandedUrl)
			}

			sb.WriteString(
				fmt.Sprintf(
					"%s\n",
					html.EscapeString(tweet),
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
	images := make([]Media, 0, len(response.Includes.Media))
	var video Media
	if len(response.Includes.Media) > 0 {
		for _, media := range response.Includes.Media {
			if media.Type == "photo" {
				images = append(images, media)
			} else if media.Type == "video" || media.Type == "animated_gif" {
				video = media
			}
		}
	}
	if len(images) == 1 { // One picture = send as preview
		sendOptions.DisableWebPagePreview = false
		return c.Reply(
			utils.EmbedImage(images[0].Url)+sb.String(),
			sendOptions,
		)
	}

	err = c.Reply(sb.String(), sendOptions)
	if err != nil {
		return err
	}

	// Send video as seperate message
	if video.MediaKey != "" {
		_ = c.Notify(telebot.UploadingVideo)

		// Need to contact 1.1 API since v2 API doesn't return direct URL to video
		//	See: https://twitterdevfeedback.uservoice.com/forums/930250-/suggestions/41291761-
		method := "GET"
		api11Url := fmt.Sprintf("https://api.twitter.com/1.1/statuses/show.json?id=%s&include_entities=true&trim_user=true&tweet_mode=extended",
			response.Tweet.ID)

		auth := OAuth1{
			ConsumerKey:    p.consumerKey,
			ConsumerSecret: p.consumerSecret,
			AccessToken:    p.accessToken,
			AccessSecret:   p.accessSecret,
		}

		authHeader := auth.BuildOAuth1Header(method, api11Url, map[string]string{
			"id":               response.Tweet.ID,
			"include_entities": "true",
			"trim_user":        "true",
			"tweet_mode":       "extended",
		})

		var videoResponse Response11
		err := utils.GetRequestWithHeader(
			api11Url,
			map[string]string{"Authorization": authHeader},
			&videoResponse,
		)
		if err != nil {
			log.Err(err).Str("url", api11Url).Msg("Error while contacting v1.1 API")
			return nil
		}

		if len(videoResponse.ExtendedEntities.Media) == 0 {
			log.Error().Str("url", api11Url).Msg("No video found")
			return nil
		}

		videoUrl := videoResponse.ExtendedEntities.Media[0].HighestResolution()
		if videoUrl == "" {
			log.Error().Str("url", api11Url).Msg("No video URL found")
			return nil
		}

		caption := videoUrl
		if video.PublicMetrics.ViewCount > 0 {
			plural := ""
			if video.PublicMetrics.ViewCount != 1 {
				plural = "e"
			}
			caption = fmt.Sprintf(
				"%s (%s Aufruf%s)",
				videoUrl,
				utils.FormatThousand(video.PublicMetrics.ViewCount),
				plural,
			)
		}

		err = c.Reply(
			&telebot.Video{
				File:      telebot.FromURL(videoUrl),
				Caption:   caption,
				Streaming: true,
			},
			telebot.Silent,
			telebot.AllowWithoutReply,
		)

		if err != nil {
			// Sending failed -send video manually
			log.Err(err).Str("url", videoUrl).Msg("Error while sending video through telegram; downloading")
			msg, err := c.Bot().Reply(c.Message(),
				fmt.Sprintf(
					"<i>🕒 <a href=\"%s\">Video</a> wird heruntergeladen und gesendet...</i>",
					videoUrl,
				),
				utils.DefaultSendOptions,
			)
			if err != nil {
				// This would be very awkward
				log.Err(err).Msg("Could not send initial 'download video' message")
			}

			_ = c.Notify(telebot.UploadingVideo)

			resp, err := http.Get(videoUrl)
			log.Info().Str("url", videoUrl).Msg("Downloading video")
			if err != nil {
				// Downloading failed - send the video URL as text
				log.Err(err).Str("url", videoUrl).Msg("Error while downloading video")
				err := c.Reply(videoUrl, telebot.Silent, telebot.AllowWithoutReply)
				if err != nil {
					log.Err(err).Str("url", videoUrl).Msg("Error while replying with video link")
				}
				_ = c.Bot().Delete(msg)
				return nil
			}

			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Err(err).Msg("Error while closing video body")
				}
			}(resp.Body)

			err = c.Reply(
				&telebot.Video{
					File:      telebot.FromReader(resp.Body),
					Caption:   caption,
					Streaming: true,
				},
				telebot.Silent,
			)
			if err != nil {
				// Last resort: Send video URL as text
				log.Err(err).Str("url", videoUrl).Msg("Error while replying with downloaded video")
				err := c.Reply(videoUrl, telebot.Silent, telebot.AllowWithoutReply)
				if err != nil {
					log.Err(err).Str("url", videoUrl).Msg("Error while replying with video link")
				}
			}
			_ = c.Bot().Delete(msg)
		}

		return nil
	}

	// Send images (> 1) as seperate message (album)
	if len(images) > 1 {
		_ = c.Notify(telebot.UploadingPhoto)
		album := make([]telebot.Inputtable, 0, len(images))
		for _, image := range images {
			album = append(album, &telebot.Photo{File: telebot.FromURL(image.Url)})
		}

		err := c.SendAlbum(album, telebot.Silent)
		if err != nil {
			// Group send failed - sending images manually as seperate messages
			log.Err(err).Msg("Error while sending album")
			for _, image := range images {
				_ = c.Notify(telebot.UploadingPhoto)

				func() {
					resp, err := http.Get(image.Url)
					log.Info().Str("url", image.Url).Msg("Downloading image")
					if err != nil {
						log.Err(err).Str("url", image.Url).Msg("Error while downloading image")
						err := c.Reply(image.Url, telebot.Silent, telebot.AllowWithoutReply)
						if err != nil {
							log.Err(err).Str("url", image.Url).Msg("Error while replying with image link")
						}
						return
					}

					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							log.Err(err).Msg("Error while closing image body")
						}
					}(resp.Body)

					err = c.Reply(&telebot.Photo{File: telebot.FromReader(resp.Body)}, telebot.Silent)
					if err != nil {
						// Last resort: Send image URL as text
						log.Err(err).Str("url", image.Url).Msg("Error while replying with downloaded image")
						err := c.Reply(image.Url, telebot.Silent, telebot.AllowWithoutReply)
						if err != nil {
							log.Err(err).Str("url", image.Url).Msg("Error while sending image link")
						}
					}
				}()
			}
		}
	}

	return nil
}
