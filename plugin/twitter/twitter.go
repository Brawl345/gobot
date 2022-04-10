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

var log = logger.NewLogger("twitter")

type (
	Plugin struct {
		bearerToken string
	}
)

func New(credentialsService models.CredentialService) *Plugin {
	bearerToken, err := credentialsService.GetKey("twitter_bearer_token")

	if err != nil {
		log.Warn().Msg("twitter_bearer_token not found")
	}

	return &Plugin{
		bearerToken: bearerToken,
	}
}

func doTwitterRequest(url string, bearerToken string, result any) error {
	log.Debug().Str("url", url).Send()
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

	defer resp.Body.Close()
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
		return &twitterError
	}

	var partialError PartialError

	err = json.Unmarshal(body, &partialError)
	if err == nil && partialError.Errors != nil {
		return &partialError
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

func (*Plugin) Name() string {
	return "twitter"
}

func (plg *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("twitter\\.com/[0-9A-Za-z_]+/status(?:es)?/(\\d+)"),
			HandlerFunc: plg.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("twitter\\.com/status(?:es)?/(\\d+)"),
			HandlerFunc: plg.OnStatus,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("nitter\\.net/[0-9A-Za-z_]+/status(?:es)?/(\\d+)"),
			HandlerFunc: plg.OnStatus,
		},
	}
}

func (plg *Plugin) OnStatus(c plugin.NextbotContext) error {
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
		"alt_text,public_metrics,type,url",
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
		plg.bearerToken,
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
		} else if errors.As(err, &partialError) { // // Log only errors that are not "status not found"
			for _, err := range partialError.Errors {
				if err.Title == "Not Found Error" {
					return c.Reply("❌ Der Status wurde nicht gefunden.")
				}
				if err.Title == "Authorization Error" {
					return c.Reply("❌ Die Tweets dieses Nutzers sind privat.")
				}
			}
			log.Error().Interface("error", partialError.Errors).Send()
			return c.Reply(fmt.Sprintf("❌ <b>API-Fehler:</b> %s", html.EscapeString(partialError.Errors[0].Detail)),
				utils.DefaultSendOptions)
		} else {
			log.Err(err).Send()
		}
		return c.Reply("❌ Bei der Anfrage ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	log.Debug().Interface("response", response).Send()

	sendOptions := &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
	}
	var sb strings.Builder

	author := response.Includes.User(response.Tweet.AuthorID)
	sb.WriteString(fmt.Sprintf("%s\n", author.String()))

	if response.Tweet.Text != "" && !(response.Tweet.Withheld.InGermany() && response.Tweet.Withheld.Scope == "tweet") {
		tweet := response.Tweet.Text
		for _, entityURL := range response.Tweet.Entities.URLs {
			if strings.Contains(entityURL.ExpandedUrl, response.Tweet.ID) {
				tweet = strings.ReplaceAll(tweet, entityURL.Url, "")
			} else {
				tweet = strings.ReplaceAll(tweet, entityURL.Url, entityURL.ExpandedUrl)
			}
		}

		sb.WriteString(
			fmt.Sprintf(
				"%s\n",
				html.EscapeString(tweet),
			),
		)
	}

	if response.Tweet.Withheld.InGermany() {
		sb.WriteString(fmt.Sprintf("%s\n", response.Tweet.Withheld.String()))
	}

	timezone := utils.GermanTimezone()
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
					"%d) %s <i>(%d Stimme%s, %.1f %%)</i>\n",
					option.Position,
					html.EscapeString(option.Label),
					option.Votes,
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
				"\n<i>%d Stimme%s - endet%s am %s</i>\n\n",
				totalVotes,
				plural,
				closed,
				poll.EndDatetime.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
			),
		)

	}

	sb.WriteString(
		fmt.Sprintf(
			"📅 %s",
			response.Tweet.CreatedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
		),
	)

	sb.WriteString(response.Tweet.PublicMetrics.String())

	images := make([]Media, 0, len(response.Includes.Media))
	var video Media
	if len(response.Includes.Media) > 0 {
		for _, media := range response.Includes.Media {
			if media.Type == "photo" {
				images = append(images, media)
			} else if media.Type == "video" {
				video = media
			}
		}
	}

	quote := response.Quote()
	if quote != nil {
		sb.WriteString("\n\n")
		quoteAuthor := response.Includes.User(quote.AuthorID)
		sb.WriteString(
			fmt.Sprintf(
				"<b>Zitat von</b> %s\n",
				quoteAuthor.String(),
			),
		)

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

			if quote.Withheld.InGermany() {
				sb.WriteString(fmt.Sprintf("%s\n", quote.Withheld.String()))
			}

			if len(quote.Attachments.PollIDs) > 0 {
				sb.WriteString(
					fmt.Sprintf(
						"📊 <i>Dieser Tweet enthält eine Umfrage - <a href=\"https://twitter.com/%s/status/%s\">rufe ihn im Browser auf</a>, um sie anzuzeigen</i>\n",
						quoteAuthor.Username,
						quote.ID,
					),
				)
			}

			sb.WriteString(
				fmt.Sprintf(
					"📅 %s",
					quote.CreatedAt.In(timezone).Format("02.01.2006, 15:04:05 Uhr"),
				),
			)

			sb.WriteString(quote.PublicMetrics.String())

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

	// TODO: contact 1.1 API
	if video.Url != "" {
		// TODO
	}

	if len(images) > 1 { // Multiple pictures = send as album
		c.Notify(telebot.UploadingPhoto)
		album := make([]telebot.Inputtable, 0, len(images))
		for _, image := range images {
			album = append(album, &telebot.Photo{File: telebot.FromURL(image.Url)})
		}
		err := c.SendAlbum(album, telebot.Silent)
		if err != nil {
			log.Err(err).Msg("Error while sending album")
			// Group send failed - sending images manually
			for _, image := range images {
				c.Notify(telebot.UploadingPhoto)

				func() {
					resp, err := http.Get(image.Url)
					log.Info().Str("url", image.Url).Msg("Downloading image")
					if err != nil {
						log.Err(err).Str("url", image.Url).Send()
						err := c.Reply(image.Url, telebot.Silent, telebot.AllowWithoutReply)
						if err != nil {
							log.Err(err).Str("url", image.Url).Msg("Error while sending image link")
						}
						return
					}

					defer resp.Body.Close()

					err = c.Reply(&telebot.Photo{File: telebot.FromReader(resp.Body)}, telebot.Silent)
					if err != nil {
						// Last resort: Send image URL as text
						log.Err(err).Str("url", image.Url).Send()
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
