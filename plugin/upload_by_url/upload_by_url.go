package upload_by_url

import (
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/exp/slices"
)

var (
	log      = logger.New("upload_by_url")
	audioExt = []string{"mp3", "ogg", "ogv", "flac", "wav"}
	imageExt = []string{"jpg", "jpeg", "png"}
	videoExt = []string{"mp4", "avi"}
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "upload_by_url"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(https?://.+\.(zip|7z|rar|tar\.(?:gz|bzip2)|jpe?g|png|gif|apk|avi|wav|mp[34]|og[gv]))`),
			HandlerFunc: onFileLink,
		},
	}
}

func onFileLink(b *gotgbot.Bot, c plugin.GobotContext) error {
	url := c.Matches[1]
	ext := c.Matches[2]

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Err(err).
			Str("url", url).
			Msg("Failed to create request")
		return nil
	}
	req.Header.Set("User-Agent", utils.UserAgent)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Err(err).
			Str("url", url).
			Msg("Failed to send request")
		return nil
	}

	fileSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		log.Err(err).
			Str("url", url).
			Msg("Failed to parse content length")
		return nil
	}
	if fileSize > tgUtils.MaxFilesizeUpload {
		log.Error().
			Str("url", url).
			Int64("fileSize", fileSize).
			Msg("File is too big")
		return nil
	}

	if fileSize == 0 {
		log.Error().
			Str("url", url).
			Msg("File is empty")
		return nil
	}

	if slices.Contains(imageExt, ext) && fileSize < tgUtils.MaxPhotosizeThroughTelegram {
		return nil
	}

	replyParams := &gotgbot.ReplyParameters{
		MessageId: c.EffectiveMessage.MessageId,
	}

	// Send file through Telegram first
	if fileSize < tgUtils.MaxFilesizeDownload {
		if slices.Contains(audioExt, ext) {
			_, err = b.SendAudio(c.EffectiveChat.Id, url, &gotgbot.SendAudioOpts{ReplyParameters: replyParams})
		} else if slices.Contains(videoExt, ext) {
			_, err = b.SendVideo(c.EffectiveChat.Id, url,
				&gotgbot.SendVideoOpts{ReplyParameters: replyParams, SupportsStreaming: true},
			)
		} else if slices.Contains(imageExt, ext) && fileSize < tgUtils.MaxPhotosizeUpload {
			_, err = b.SendPhoto(c.EffectiveChat.Id, url, &gotgbot.SendPhotoOpts{ReplyParameters: replyParams})
		} else {
			_, err = b.SendDocument(c.EffectiveChat.Id, url, &gotgbot.SendDocumentOpts{ReplyParameters: replyParams})
		}
	}

	// Send file manually
	if err != nil {
		log.Warn().
			Err(err).
			Str("url", url).
			Msg("Failed to send file through Telegram")

		resp, err := httpUtils.DefaultHttpClient.Get(url)
		if err != nil {
			log.Err(err).
				Str("url", url).
				Msg("Failed to get file")
			return nil
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Err(err).
					Str("url", url).
					Msg("Failed to close response body")
			}
		}(resp.Body)

		fileName := path.Base(url)
		file := gotgbot.NamedFile{File: resp.Body, FileName: fileName}
		if slices.Contains(audioExt, ext) {
			_, err = b.SendAudio(c.EffectiveChat.Id, file, &gotgbot.SendAudioOpts{ReplyParameters: replyParams})
		} else if slices.Contains(videoExt, ext) {
			_, err = b.SendVideo(c.EffectiveChat.Id, file,
				&gotgbot.SendVideoOpts{ReplyParameters: replyParams, SupportsStreaming: true},
			)
		} else if slices.Contains(imageExt, ext) && fileSize < tgUtils.MaxPhotosizeUpload {
			_, err = b.SendPhoto(c.EffectiveChat.Id, file, &gotgbot.SendPhotoOpts{ReplyParameters: replyParams})
			if err != nil {
				_, err = b.SendDocument(c.EffectiveChat.Id, file, &gotgbot.SendDocumentOpts{ReplyParameters: replyParams})
			}
		} else {
			_, err = b.SendDocument(c.EffectiveChat.Id, file, &gotgbot.SendDocumentOpts{ReplyParameters: replyParams})
		}
		if err != nil {
			log.Err(err).
				Str("url", url).
				Msg("Failed to send file manually")
			return nil
		}
	}
	return nil
}
