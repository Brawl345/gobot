package google_images

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
)

var log = logger.New("google_images")

type (
	Plugin struct {
		credentialService   model.CredentialService
		googleImagesService Service
	}

	CleanupService interface {
		Cleanup() error
	}

	Service interface {
		GetImages(query string) (model.GoogleImages, error)
		GetImagesFromQueryID(queryID int64) (model.GoogleImages, error)
		SaveImages(query string, wrapper *model.GoogleImages) (int64, error)
		SaveIndex(queryID int64, index int) error
	}
)

func New(credentialService model.CredentialService, googleImagesService Service, cleanupService CleanupService) *Plugin {
	time.AfterFunc(24*time.Hour, func() {
		cleanup(cleanupService)
	})

	return &Plugin{
		credentialService:   credentialService,
		googleImagesService: googleImagesService,
	}
}

func cleanup(cleanupService CleanupService) {
	log.Debug().Msg("starting cleanup")
	defer time.AfterFunc(24*time.Hour, func() {
		cleanup(cleanupService)
	})

	err := cleanupService.Cleanup()
	if err != nil {
		log.Error().Err(err).Msg("error cleaning up google images")
	}

}

func (p *Plugin) Name() string {
	return "google_images"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "i",
			Description: "<Suchbegriff> - Nach Bildern suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/i(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onImageSearch,
		},
		&plugin.CallbackHandler{
			Trigger:      regexp.MustCompile(`^i:(\d+)$`),
			HandlerFunc:  p.onImageSearchCallback,
			DeleteButton: true,
			Cooldown:     5 * time.Second,
		},
	}
}

func (p *Plugin) doImageSearch(b *gotgbot.Bot, c *plugin.GobotContext) error {
	apiKey := p.credentialService.GetKey("google_api_key")
	if apiKey == "" {
		log.Warn().Msg("google_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>google_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	searchEngineID := p.credentialService.GetKey("google_search_engine_id")
	if searchEngineID == "" {
		log.Warn().Msg("google_search_engine_id not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>google_search_engine_id</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	query := c.Matches[1]

	var wrapper model.GoogleImages
	var err error
	if c.CallbackQuery != nil {
		queryID, err := strconv.ParseInt(query, 10, 64)
		if err != nil {
			return err
		}

		wrapper, err = p.googleImagesService.GetImagesFromQueryID(queryID)
		if err != nil {
			return err
		}
		if len(wrapper.Images) == 0 {
			return ErrNoImagesFound
		}
	} else {
		wrapper, err = p.googleImagesService.GetImages(query)
	}
	if err != nil {
		return fmt.Errorf("error getting google images from db: %w", err)
	}

	if len(wrapper.Images) == 0 {
		_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)
		requestUrl := url.URL{
			Scheme: "https",
			Host:   "customsearch.googleapis.com",
			Path:   "/customsearch/v1",
		}

		q := requestUrl.Query()
		q.Set("key", apiKey)
		q.Set("cx", searchEngineID)
		q.Set("q", query)
		q.Set("hl", "de")
		q.Set("gl", "de")
		q.Set("num", "10")
		q.Set("safe", "active")
		q.Set("searchType", "image")
		q.Set("fields", "items(link,mime,image/contextLink)")

		requestUrl.RawQuery = q.Encode()

		var response Response
		err = httpUtils.MakeRequest(httpUtils.RequestOptions{
			Method:   httpUtils.MethodGet,
			URL:      requestUrl.String(),
			Response: &response,
		})

		if err != nil {
			return fmt.Errorf("error getting google images: %w", err)
		}

		if len(response.Items) == 0 {
			return ErrNoImagesFound
		}

		items := make([]model.Image, len(response.Items))
		for i, v := range response.Items {
			items[i] = v
		}

		wrapper.Images = items
		queryID, err := p.googleImagesService.SaveImages(query, &model.GoogleImages{
			Images: wrapper.Images,
		})
		if err != nil {
			return fmt.Errorf("error saving google images: %w", err)
		}
		wrapper.QueryID = queryID
	}

	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)
	index := wrapper.CurrentIndex
	var success bool
	var numberOfTries int
	var maxNumberOfTries = len(wrapper.Images)

	for !success && numberOfTries < maxNumberOfTries {
		image := wrapper.Images[index]
		caption := fmt.Sprintf(
			"<a href=\"%s\">🖼 Vollbild</a> • <a href=\"%s\">🌐 Seite aufrufen</a>",
			image.ImageLink(),
			image.ContextLink(),
		)
		replyMarkup := &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "Nächstes Bild",
						CallbackData: fmt.Sprintf("i:%d", wrapper.QueryID),
					},
				},
			},
		}

		if image.IsGIF() {
			_, err = b.SendDocument(c.EffectiveChat.Id, image.ImageLink(), &gotgbot.SendDocumentOpts{
				Caption: caption,
				ReplyParameters: &gotgbot.ReplyParameters{
					AllowSendingWithoutReply: true,
					MessageId:                c.EffectiveMessage.MessageId,
				},
				DisableNotification: true,
				ParseMode:           gotgbot.ParseModeHTML,
				ReplyMarkup:         replyMarkup,
			})
		} else {
			_, err = b.SendPhoto(c.EffectiveChat.Id, image.ImageLink(), &gotgbot.SendPhotoOpts{
				Caption: caption,
				ReplyParameters: &gotgbot.ReplyParameters{
					AllowSendingWithoutReply: true,
					MessageId:                c.EffectiveMessage.MessageId,
				},
				DisableNotification: true,
				ParseMode:           gotgbot.ParseModeHTML,
				ReplyMarkup:         replyMarkup,
			})
		}

		index++
		if index >= len(wrapper.Images) {
			index = 0
		}

		if err == nil {
			success = true
		} else {
			success = false
			numberOfTries++
			log.Err(err).
				Interface("image", image).
				Msg("error sending image")
		}
	}

	if success {
		err = p.googleImagesService.SaveIndex(wrapper.QueryID, index)
		if err != nil {
			log.Err(err).
				Msg("error saving to db")
		}
		return nil
	} else {
		return ErrCouldNotDownloadAnyImage
	}
}

func (p *Plugin) onImageSearch(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.doImageSearch(b, &c)
	var httpError *httpUtils.HttpError
	if err != nil {
		if errors.Is(err, ErrNoImagesFound) {
			_, err = c.EffectiveMessage.Reply(b, "❌ Keine Bilder gefunden.", utils.DefaultSendOptions())
		} else if errors.Is(err, ErrCouldNotDownloadAnyImage) {
			_, err = c.EffectiveMessage.Reply(b, "❌ Es konnte kein Bild heruntergeladen werden.", utils.DefaultSendOptions())
		} else if errors.As(err, &httpError) && httpError.StatusCode == http.StatusTooManyRequests {
			_, err = c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht. Bitte versuche es morgen erneut.", utils.DefaultSendOptions())
		} else {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("error doing image search")
			_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		}
		return err
	}
	return nil
}

func (p *Plugin) onImageSearchCallback(b *gotgbot.Bot, c plugin.GobotContext) error {
	// ignore callback queries older than 7 days
	callbackTime := utils.TimestampToTime(c.CallbackQuery.Message.GetDate())
	if callbackTime.Add(utils.Week).Before(time.Now()) {
		_, err := c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "❌ Bitte sende den Befehl erneut ab.",
			ShowAlert: true,
		})
		return err
	}

	_, _ = c.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text:      "Nächstes Bild wird gesendet...",
		ShowAlert: false,
	})
	err := p.doImageSearch(b, &c)
	if err != nil {
		log.Err(err).
			Str("query_id", c.Matches[1]).
			Msg("error doing image search")
	}
	return nil
}
