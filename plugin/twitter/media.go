package twitter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

const (
	maxPhotoURLSize    = 5_000_000
	maxPhotoUploadSize = 10_000_000
	maxFileURLSize     = 20_000_000
	maxFileUploadSize  = tgUtils.MaxFilesizeUpload
)

type preparedMedium struct {
	medium     Medium
	url        string
	size       int64
	uploadOnly bool
	data       []byte
	fullURL    string
	fullSize   int64
}

func (p *preparedMedium) usable() bool {
	return p.url != ""
}

func (p *preparedMedium) needsDownload() bool {
	return p.usable() && p.data == nil
}

func prepareMedia(media []Medium) []*preparedMedium {
	prepared := make([]*preparedMedium, len(media))
	var wg sync.WaitGroup
	for i, medium := range media {
		wg.Go(func() {
			prepared[i] = prepareMedium(medium)
		})
	}
	wg.Wait()

	var uploads sync.WaitGroup
	for _, p := range prepared {
		if p.uploadOnly {
			uploads.Go(func() { p.download() })
		}
	}
	uploads.Wait()
	return prepared
}

func prepareMedium(medium Medium) *preparedMedium {
	p := &preparedMedium{medium: medium}

	if medium.IsPhoto() {
		orig := medium.MediaUrlHttps + ":orig"
		size := headSize(orig)
		if p.pick(orig, size, maxPhotoURLSize, maxPhotoUploadSize) {
			return p
		}
		p.fullURL, p.fullSize = orig, size
		large := medium.MediaUrlHttps + "?name=4096x4096"
		p.pick(large, headSize(large), maxPhotoURLSize, maxPhotoUploadSize)
		return p
	}

	variants := medium.mp4Variants()
	if len(variants) == 0 {
		return p
	}
	sizes := make([]int64, len(variants))
	var wg sync.WaitGroup
	for i, variant := range variants {
		wg.Go(func() { sizes[i] = headSize(variant) })
	}
	wg.Wait()

	for i, variant := range variants {
		if p.pick(variant, sizes[i], maxFileURLSize, maxFileUploadSize) {
			if i > 0 {
				p.fullURL, p.fullSize = variants[0], sizes[0]
			}
			return p
		}
	}
	p.fullURL, p.fullSize = variants[0], sizes[0]
	return p
}

func (p *preparedMedium) pick(link string, size int64, urlLimit, uploadLimit int64) bool {
	if size > uploadLimit {
		return false
	}
	p.url, p.size = link, size
	p.uploadOnly = size > urlLimit
	return true
}

func (p *preparedMedium) download() {
	limit := int64(maxFileUploadSize)
	if p.medium.IsPhoto() {
		limit = maxPhotoUploadSize
	}
	data, err := downloadMedium(p.url, limit)
	if err != nil {
		log.Err(err).Str("url", p.url).Msg("Error while downloading")
		if p.fullURL == "" {
			p.fullURL, p.fullSize = p.url, p.size
		}
		p.url = ""
		return
	}
	p.data = data
}

func downloadMissing(prepared []*preparedMedium) {
	var wg sync.WaitGroup
	for _, p := range prepared {
		if p.needsDownload() {
			wg.Go(p.download)
		}
	}
	wg.Wait()
}

func (m *Medium) mp4Variants() []string {
	type variant struct {
		url     string
		bitrate int
	}
	var variants []variant
	for _, v := range m.VideoInfo.Variants {
		if strings.Contains(v.Url, ".mp4") {
			variants = append(variants, variant{v.Url, v.Bitrate})
		}
	}
	slices.SortStableFunc(variants, func(a, b variant) int { return b.bitrate - a.bitrate })
	urls := make([]string, len(variants))
	for i, v := range variants {
		urls[i] = v.url
	}
	return urls
}

func headSize(link string) int64 {
	resp, err := httpUtils.DefaultHttpClient.Head(link)
	if err != nil {
		return -1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return -1
	}
	return resp.ContentLength
}

func downloadMedium(link string, limit int64) ([]byte, error) {
	log.Info().Str("url", link).Msg("Downloading")
	resp, err := httpUtils.DefaultHttpClient.Get(link)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Error while closing body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("file too large")
	}
	return data, nil
}

func sendAlbums(b *gotgbot.Bot, c plugin.GobotContext, media []Medium, spoiler bool) {
	if len(media) == 0 {
		return
	}
	albumCount := (len(media) + tgUtils.MaxAlbumSize - 1) / tgUtils.MaxAlbumSize
	albumSize := (len(media) + albumCount - 1) / albumCount
	for chunk := range slices.Chunk(media, albumSize) {
		sendAlbum(b, c, chunk, spoiler)
	}
}

func sendAlbum(b *gotgbot.Bot, c plugin.GobotContext, media []Medium, spoiler bool) {
	if len(media) > 1 {
		files := make([]gotgbot.InputFileOrString, len(media))
		for i, medium := range media {
			files[i] = medium.InputFile()
		}

		err := replyAlbum(b, c, media, files, spoiler)
		if err == nil {
			return
		}
		log.Err(err).Msg("Error while sending album")
	}

	msg, err := c.EffectiveMessage.ReplyMessage(b,
		"<i>🕒 Medien werden heruntergeladen und gesendet...</i>",
		utils.DefaultSendOptions(),
	)
	if err != nil {
		log.Err(err).Msg("Could not send initial 'download' message")
	}
	defer func() {
		if msg != nil {
			_, _ = msg.Delete(b, nil)
		}
	}()

	downloads := downloadMedia(media)

	downloaded := make([]Medium, 0, len(media))
	files := make([]gotgbot.InputFileOrString, 0, len(media))
	failed := make([]Medium, 0, len(media))
	for i, medium := range media {
		if downloads[i] == nil {
			failed = append(failed, medium)
			continue
		}
		downloaded = append(downloaded, medium)
		files = append(files, gotgbot.InputFileByReader(medium.IdStr, bytes.NewReader(downloads[i])))
	}

	albumSent := false
	if len(downloaded) > 1 {
		err := replyAlbum(b, c, downloaded, files, spoiler)
		if err == nil {
			albumSent = true
		} else {
			log.Err(err).Msg("Error while sending album with downloaded media")
		}
	}

	if !albumSent {
		for i, medium := range downloaded {
			err := replyMedium(b, c, medium, files[i], spoiler)
			if err != nil {
				log.Err(err).Str("url", medium.Link()).Msg("Error while replying with downloaded medium")
				failed = append(failed, medium)
			}
		}
	}

	for _, medium := range failed {
		_, err := c.EffectiveMessage.ReplyMessage(b, medium.Caption(), &gotgbot.SendMessageOpts{
			ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
			DisableNotification: true,
		})
		if err != nil {
			log.Err(err).Str("url", medium.Link()).Msg("Error while sending medium link")
		}
	}
}

func replyAlbum(b *gotgbot.Bot, c plugin.GobotContext, media []Medium, files []gotgbot.InputFileOrString, spoiler bool) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)
	album := make([]gotgbot.InputMedia, 0, len(media))
	for i, medium := range media {
		if medium.IsPhoto() {
			album = append(album, gotgbot.InputMediaPhoto{Caption: medium.Caption(), Media: files[i], HasSpoiler: spoiler})
		} else {
			album = append(album, gotgbot.InputMediaVideo{Caption: medium.Caption(), Media: files[i], SupportsStreaming: true, HasSpoiler: spoiler})
		}
	}

	_, err := c.EffectiveMessage.ReplyMediaGroup(b, album, &gotgbot.SendMediaGroupOpts{DisableNotification: true})
	return err
}

func replyMedium(b *gotgbot.Bot, c plugin.GobotContext, medium Medium, file gotgbot.InputFileOrString, spoiler bool) error {
	if medium.IsPhoto() {
		_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)
		_, err := c.EffectiveMessage.ReplyPhoto(b, file, &gotgbot.SendPhotoOpts{
			ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
			DisableNotification: true,
			HasSpoiler:          spoiler,
		})
		return err
	}

	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadVideo, nil)
	_, err := c.EffectiveMessage.ReplyVideo(b, file, &gotgbot.SendVideoOpts{
		Caption:             medium.Caption(),
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		DisableNotification: true,
		SupportsStreaming:   true,
		HasSpoiler:          spoiler,
	})
	return err
}

func downloadMedia(media []Medium) [][]byte {
	results := make([][]byte, len(media))
	var wg sync.WaitGroup
	for i, medium := range media {
		wg.Go(func() {
			data, err := downloadMedium(medium.Link(), maxFileUploadSize)
			if err != nil {
				log.Err(err).Str("url", medium.Link()).Msg("Error while downloading")
				return
			}
			results[i] = data
		})
	}
	wg.Wait()
	return results
}
