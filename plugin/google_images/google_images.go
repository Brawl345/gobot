package google_images

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"gopkg.in/telebot.v3"
)

type (
	Plugin struct {
		apiKey              string
		searchEngineID      string
		googleImagesService Service
	}

	CleanupService interface {
		Cleanup() error
	}

	Service interface {
		GetImages(query string) (models.GoogleImages, error)
		GetImagesFromQueryID(queryID int64) (models.GoogleImages, error)
		SaveImages(query string, wrapper *models.GoogleImages) (int64, error)
		SaveIndex(queryID int64, index int) error
	}
)

func New(credentialService models.CredentialService, googleImagesService Service, cleanupService CleanupService) *Plugin {
	apiKey, err := credentialService.GetKey("google_api_key")
	if err != nil {
		log.Warn().Msg("google_api_key not found")
	}

	searchEngineID, err := credentialService.GetKey("google_search_engine_id")
	if err != nil {
		log.Warn().Msg("google_search_engine_id not found")
	}

	time.AfterFunc(24*time.Hour, func() {
		cleanup(cleanupService)
	})

	return &Plugin{
		apiKey:              apiKey,
		searchEngineID:      searchEngineID,
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "i",
			Description: "<Suchbegriff> - Nach Bildern suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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

func (p *Plugin) doImageSearch(c *plugin.GobotContext) error {
	query := c.Matches[1]

	var wrapper models.GoogleImages
	var err error
	if c.Callback() != nil {
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
		_ = c.Notify(telebot.UploadingPhoto)
		requestUrl := url.URL{
			Scheme: "https",
			Host:   "customsearch.googleapis.com",
			Path:   "/customsearch/v1",
		}

		q := requestUrl.Query()
		q.Set("key", p.apiKey)
		q.Set("cx", p.searchEngineID)
		q.Set("q", query)
		q.Set("hl", "de")
		q.Set("gl", "de")
		q.Set("num", "10")
		q.Set("safe", "active")
		q.Set("searchType", "image")
		q.Set("fields", "items(link,mime,image/contextLink)")

		requestUrl.RawQuery = q.Encode()

		var response Response
		err = utils.GetRequest(requestUrl.String(), &response)

		if err != nil {
			return fmt.Errorf("error getting google images: %w", err)
		}

		if len(response.Items) == 0 {
			return ErrNoImagesFound
		}

		items := make([]models.Image, len(response.Items))
		for i, v := range response.Items {
			items[i] = v
		}

		wrapper.Images = items
		queryID, err := p.googleImagesService.SaveImages(query, &models.GoogleImages{
			Images: wrapper.Images,
		})
		if err != nil {
			return fmt.Errorf("error saving google images: %w", err)
		}
		wrapper.QueryID = queryID
	}

	_ = c.Notify(telebot.UploadingPhoto)
	index := wrapper.CurrentIndex
	var success bool
	var numberOfTries int
	var maxNumberOfTries = len(wrapper.Images)
	imageSendOptions := &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
		ReplyMarkup: &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{
					{
						Text: "Nächstes Bild",
						Data: fmt.Sprintf("i:%d", wrapper.QueryID),
					},
				},
			},
		},
	}

	for !success && numberOfTries < maxNumberOfTries {
		image := wrapper.Images[index]
		caption := fmt.Sprintf(
			"<a href=\"%s\">🖼 Vollbild</a> • <a href=\"%s\">🌐 Seite aufrufen</a>",
			image.ImageLink(),
			image.ContextLink(),
		)

		if image.IsGIF() {
			err = c.Reply(&telebot.Document{
				File:    telebot.FromURL(image.ImageLink()),
				Caption: caption,
			}, imageSendOptions)
		} else {
			err = c.Reply(&telebot.Photo{
				File:    telebot.FromURL(image.ImageLink()),
				Caption: caption,
			}, imageSendOptions)
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

func (p *Plugin) onImageSearch(c plugin.GobotContext) error {
	err := p.doImageSearch(&c)
	var httpError *utils.HttpError
	if err != nil {
		if errors.Is(err, ErrNoImagesFound) {
			return c.Reply("❌ Keine Bilder gefunden.", utils.DefaultSendOptions)
		} else if err == ErrCouldNotDownloadAnyImage {
			return c.Reply("❌ Es konnte kein Bild heruntergeladen werden.", utils.DefaultSendOptions)
		} else if errors.As(err, &httpError) && httpError.StatusCode == 429 {
			return c.Reply("❌ Rate-Limit erreicht. Bitte versuche es morgen erneut.", utils.DefaultSendOptions)
		} else {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("error doing image search")
			return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		}
	}
	return nil
}

func (p *Plugin) onImageSearchCallback(c plugin.GobotContext) error {
	// ignore callback queries older than 7 days
	if c.Callback().Message.Time().Add(utils.Week).Before(time.Now()) {
		return c.Respond(&telebot.CallbackResponse{
			Text:      "❌ Bitte sende den Befehl erneut ab.",
			ShowAlert: true,
		})
	}

	_ = c.Respond(&telebot.CallbackResponse{
		Text:      "Nächstes Bild wird gesendet...",
		ShowAlert: false,
	})
	err := p.doImageSearch(&c)
	if err != nil {
		log.Err(err).
			Str("query_id", c.Matches[1]).
			Msg("error doing image search")
	}
	return nil
}
