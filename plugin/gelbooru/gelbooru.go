package gelbooru

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var (
	log            = logger.New("gelbooru")
	additionalTags = []string{"sort:random", "-photorealistic"}
)

type (
	Plugin struct {
		credentialService model.CredentialService
		gelbooruService   model.GelbooruService
	}

	CleanupService interface {
		Cleanup() error
	}
)

func New(credentialService model.CredentialService, gelbooruService model.GelbooruService, cleanupService CleanupService) *Plugin {
	time.AfterFunc(24*time.Hour, func() {
		cleanup(cleanupService)
	})

	return &Plugin{
		credentialService: credentialService,
		gelbooruService:   gelbooruService,
	}
}

func cleanup(cleanupService CleanupService) {
	log.Debug().Msg("starting cleanup")
	defer time.AfterFunc(24*time.Hour, func() {
		cleanup(cleanupService)
	})

	err := cleanupService.Cleanup()
	if err != nil {
		log.Error().Err(err).Msg("error cleaning up gelbooru queries")
	}
}

func (p *Plugin) Name() string {
	return "gelbooru"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "gel",
			Description: "<Suchbegriff> - Sucht auf Gelbooru",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/gel(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onGelbooruSearch,
			GroupOnly:   false,
		},
		&plugin.CallbackHandler{
			Trigger:      regexp.MustCompile(`^gel:(\d+)$`),
			HandlerFunc:  p.onGelbooruCallback,
			DeleteButton: true,
			Cooldown:     3 * time.Second,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)gelbooru\.com/index\.php\?page=post&s=view&id=(\d+)`),
			HandlerFunc: p.onGelbooruLink,
		},
	}
}

func (p *Plugin) onGelbooruSearch(b *gotgbot.Bot, c plugin.GobotContext) error {
	query := c.Matches[1]
	return p.doGelbooruSearch(b, &c, query)
}

func (p *Plugin) onGelbooruCallback(b *gotgbot.Bot, c plugin.GobotContext) error {
	callbackTime := utils.TimestampToTime(c.CallbackQuery.Message.GetDate())
	if callbackTime.Add(utils.Week).Before(time.Now()) {
		_, err := c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "❌ Bitte sende den Befehl erneut ab.",
			ShowAlert: true,
		})
		return err
	}

	queryID, err := strconv.ParseInt(c.Matches[1], 10, 64)
	if err != nil {
		return err
	}

	query, err := p.gelbooruService.GetQuery(queryID)
	if err != nil {
		if errors.Is(err, model.ErrQueryNotFound) {
			_, err := c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "❌ Bitte sende den Befehl erneut ab.",
				ShowAlert: true,
			})
			return err
		}
		log.Err(err).Int64("query_id", queryID).Msg("error getting gelbooru query")
		_, err := c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "❌ Ein Fehler ist aufgetreten.",
			ShowAlert: true,
		})
		return err
	}

	_, _ = c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text:      "Suche erneut...",
		ShowAlert: false,
	})

	err = p.doGelbooruSearch(b, &c, query)
	if err != nil {
		log.Err(err).Str("query", query).Msg("error in search from callback")
	}
	return nil
}

func (p *Plugin) onGelbooruLink(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)
	id := c.Matches[1]

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "gelbooru.com",
		Path:   "/index.php",
	}

	q := requestUrl.Query()
	q.Set("page", "dapi")
	q.Set("q", "index")
	q.Set("s", "post")
	q.Set("json", "1")
	q.Set("id", id)

	requestUrl.RawQuery = q.Encode()

	response, err := p.fetchPost(b, &c, requestUrl)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error making gelbooru request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Post) == 0 {
		return nil
	}

	post := response.Post[0]
	return p.sendPost(b, &c, &post, nil)
}

func (p *Plugin) doGelbooruSearch(b *gotgbot.Bot, c *plugin.GobotContext, query string) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)

	fullQuery := strings.Join(append([]string{query}, additionalTags...), " ")
	if !strings.Contains(fullQuery, "rating:explicit") && !strings.Contains(fullQuery, "rating:e") {
		fullQuery += " -rating:explicit"
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "gelbooru.com",
		Path:   "/index.php",
	}

	q := requestUrl.Query()
	q.Set("page", "dapi")
	q.Set("q", "index")
	q.Set("s", "post")
	q.Set("json", "1")
	q.Set("limit", "1")
	q.Set("tags", fullQuery)

	requestUrl.RawQuery = q.Encode()

	response, err := p.fetchPost(b, c, requestUrl)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error making gelbooru request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Post) == 0 {
		return nil
	}

	queryID, err := p.gelbooruService.SaveQuery(query)
	if err != nil {
		log.Err(err).Msg("error saving gelbooru query")
		queryID = 0
	}

	post := response.Post[0]

	var replyMarkup gotgbot.InlineKeyboardMarkup
	if response.Attributes.Count > 1 && queryID > 0 {
		replyMarkup = gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "Nochmal suchen",
						CallbackData: fmt.Sprintf("gel:%d", queryID),
					},
				},
			},
		}
	}

	return p.sendPost(b, c, &post, replyMarkup)
}

func (p *Plugin) fetchPost(b *gotgbot.Bot, c *plugin.GobotContext, requestUrl url.URL) (Response, error) {
	apiKey := p.credentialService.GetKey("gelbooru_api_key")
	if apiKey == "" {
		log.Warn().Msg("gelbooru_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>gelbooru_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return Response{}, err
	}

	userId := p.credentialService.GetKey("gelbooru_user_id")
	if userId == "" {
		log.Warn().Msg("gelbooru_user_id not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>gelbooru_user_id</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return Response{}, err
	}

	q := requestUrl.Query()
	q.Set("api_key", apiKey)
	q.Set("user_id", userId)

	requestUrl.RawQuery = q.Encode()

	var response Response
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method: httpUtils.MethodGet,
		URL:    requestUrl.String(),
		Headers: map[string]string{
			"User-Agent": "Gobot/1.0 (Telegram Bot; +https://github.com/Brawl345/gobot)",
		},
		Response: &response,
	})

	if err != nil {
		return Response{}, err
	}

	if len(response.Post) == 0 {
		_, err := c.EffectiveMessage.Reply(b,
			"❌ Nichts gefunden.",
			utils.DefaultSendOptions(),
		)
		return Response{}, err
	}

	return response, nil
}

func (p *Plugin) downloadAndSend(b *gotgbot.Bot, c *plugin.GobotContext, post *Post, replyMarkup gotgbot.ReplyMarkup) error {
	fileURL := post.FileURL()
	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", "https://gelbooru.com/")
	req.Header.Set("User-Agent", "Gobot/1.0 (Telegram Bot; +https://github.com/Brawl345/gobot)")

	resp, err := httpUtils.DefaultHttpClient.Do(req)
	if err != nil {
		return err
	}
	defer func(body io.ReadCloser) {
		if closeErr := body.Close(); closeErr != nil {
			log.Err(closeErr).Msg("failed to close gelbooru response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gelbooru download: HTTP %d", resp.StatusCode)
	}

	file := gotgbot.InputFileByReader(path.Base(fileURL), resp.Body)

	if post.IsImage() {
		_, err = b.SendPhoto(c.EffectiveChat.Id, file, &gotgbot.SendPhotoOpts{
			Caption: post.Caption(),
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
				MessageId:                c.EffectiveMessage.MessageId,
			},
			DisableNotification: true,
			ReplyMarkup:         replyMarkup,
			ParseMode:           gotgbot.ParseModeHTML,
			HasSpoiler:          post.IsNSFW(),
		})
	} else if post.IsVideo() {
		_, err = b.SendVideo(c.EffectiveChat.Id, file, &gotgbot.SendVideoOpts{
			Caption: post.Caption(),
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
				MessageId:                c.EffectiveMessage.MessageId,
			},
			DisableNotification: true,
			ReplyMarkup:         replyMarkup,
			ParseMode:           gotgbot.ParseModeHTML,
			HasSpoiler:          post.IsNSFW(),
		})
	} else if post.IsGIF() {
		_, err = b.SendAnimation(c.EffectiveChat.Id, file, &gotgbot.SendAnimationOpts{
			Caption: post.Caption(),
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
				MessageId:                c.EffectiveMessage.MessageId,
			},
			DisableNotification: true,
			ReplyMarkup:         replyMarkup,
			ParseMode:           gotgbot.ParseModeHTML,
			HasSpoiler:          post.IsNSFW(),
		})
	} else {
		if post.IsNSFW() {
			_, err = b.SendMessage(c.EffectiveChat.Id, post.AltCaption(), &gotgbot.SendMessageOpts{
				ReplyParameters: &gotgbot.ReplyParameters{
					AllowSendingWithoutReply: true,
					MessageId:                c.EffectiveMessage.MessageId,
				},
				DisableNotification: true,
				ParseMode:           gotgbot.ParseModeHTML,
				ReplyMarkup:         replyMarkup,
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
			})
			return err
		}
		_, err = b.SendDocument(c.EffectiveChat.Id, file, &gotgbot.SendDocumentOpts{
			Caption: post.PostURL(),
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
				MessageId:                c.EffectiveMessage.MessageId,
			},
			DisableNotification: true,
			ParseMode:           gotgbot.ParseModeHTML,
			ReplyMarkup:         replyMarkup,
		})
	}
	return err
}

func (p *Plugin) sendPost(b *gotgbot.Bot, c *plugin.GobotContext, post *Post, replyMarkup gotgbot.ReplyMarkup) error {
	err := p.downloadAndSend(b, c, post, replyMarkup)
	if err != nil {
		_, err = b.SendMessage(c.EffectiveChat.Id, post.AltCaption(), &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: true,
				MessageId:                c.EffectiveMessage.MessageId,
			},
			DisableNotification: true,
			ParseMode:           gotgbot.ParseModeHTML,
			ReplyMarkup:         replyMarkup,
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled:       post.IsNSFW(),
				PreferLargeMedia: true,
				Url:              post.FileURL(),
				ShowAboveText:    true,
			},
		})
	}
	return err
}
