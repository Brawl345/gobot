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
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

var (
	log      = logger.New("upload_by_url")
	audioExt = []string{"mp3", "ogg", "ogv", "flac"}
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

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("(https?://.+\\.(zip|7z|rar|tar\\.(?:gz|bzip2)|jpe?g|png|gif|apk|avi|mp[34]|webp|og[gv]))"),
			HandlerFunc: onFileLink,
		},
	}
}

func onFileLink(c plugin.GobotContext) error {
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
	if fileSize > utils.MaxFilesizeUpload {
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

	if slices.Contains(imageExt, ext) && fileSize < utils.MaxPhotosizeThroughTelegram {
		return nil
	}

	// Send file through Telegram first
	if fileSize < utils.MaxFilesizeDownload {
		file := telebot.FromURL(url)

		if slices.Contains(audioExt, ext) {
			err = c.Reply(&telebot.Audio{
				File: file,
			})
		} else if slices.Contains(videoExt, ext) {
			err = c.Reply(&telebot.Video{
				File:      file,
				Streaming: true,
			})
		} else if slices.Contains(imageExt, ext) && fileSize < utils.MaxPhotosizeUpload {
			err = c.Reply(&telebot.Photo{
				File: file,
			})
		} else {
			err = c.Reply(&telebot.Document{
				File: file,
			})
		}
	}

	// Send file manually
	if err != nil {
		log.Warn().
			Err(err).
			Str("url", url).
			Msg("Failed to send file through Telegram")

		resp, err := http.Get(url)
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

		file := telebot.FromReader(resp.Body)
		fileName := path.Base(url)
		if slices.Contains(audioExt, ext) {
			err = c.Reply(&telebot.Audio{
				File:     file,
				FileName: fileName,
			})
		} else if slices.Contains(videoExt, ext) {
			err = c.Reply(&telebot.Video{
				File:      file,
				FileName:  fileName,
				Streaming: true,
			})
		} else if slices.Contains(imageExt, ext) && fileSize < utils.MaxPhotosizeUpload {
			err = c.Reply(&telebot.Photo{
				File: file,
			})
			if err != nil {
				err = c.Reply(&telebot.Document{
					File:     file,
					FileName: fileName,
				})
			}
		} else {
			err = c.Reply(&telebot.Document{
				File:     file,
				FileName: fileName,
			})
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
