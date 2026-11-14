package twitter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

const (
	maxPhotoURLSize    = 5_000_000
	maxPhotoUploadSize = 10_000_000
	maxFileURLSize     = 20_000_000
	maxFileUploadSize  = tgUtils.MaxFilesizeUpload
	// Telegram rejects the whole request with "Request Entity Too Large" above roughly 50 MB.
	maxRequestUploadSize = 49_000_000
)

type mediaVariant struct {
	url  string
	size int64
}

type preparedMedium struct {
	medium   Medium
	position int
	variants []mediaVariant
	url      string
	size     int64
	upload   bool
	data     []byte
	fullURL  string
	fullSize int64
}

func (p *preparedMedium) usable() bool {
	return p.url != "" && (!p.upload || p.data != nil)
}

func (p *preparedMedium) limits() (int64, int64) {
	if p.medium.IsPhoto() {
		return maxPhotoURLSize, maxPhotoUploadSize
	}
	return maxFileURLSize, maxFileUploadSize
}

func (p *preparedMedium) choose(maxSize int64) bool {
	urlLimit, uploadLimit := p.limits()
	for i, v := range p.variants {
		if v.size == sizeMissing || v.size > uploadLimit || v.size > maxSize {
			continue
		}
		p.url, p.size = v.url, v.size
		p.upload = v.size > urlLimit
		p.data = nil
		p.fullURL, p.fullSize = "", 0
		if i > 0 {
			p.fullURL, p.fullSize = p.variants[0].url, p.variants[0].size
		}
		return true
	}
	p.url, p.size, p.upload, p.data = "", 0, false, nil
	if len(p.variants) > 0 {
		p.fullURL, p.fullSize = p.variants[0].url, p.variants[0].size
	}
	return false
}

func (p *preparedMedium) chooseURL() bool {
	urlLimit, _ := p.limits()
	return p.choose(urlLimit)
}

func (p *preparedMedium) download() error {
	_, uploadLimit := p.limits()
	data, err := downloadMedium(p.url, uploadLimit)
	if err != nil {
		return err
	}
	p.data = data
	return nil
}

func prepareMedia(media []Medium) []*preparedMedium {
	prepared := make([]*preparedMedium, len(media))
	var wg sync.WaitGroup
	for i, medium := range media {
		p := &preparedMedium{medium: medium, position: i + 1}
		prepared[i] = p
		for _, link := range medium.variantURLs() {
			p.variants = append(p.variants, mediaVariant{url: link})
		}
		for j := range p.variants {
			wg.Go(func() { p.variants[j].size = headSize(p.variants[j].url) })
		}
	}
	wg.Wait()

	budget := int64(maxRequestUploadSize)
	for _, p := range prepared {
		p.choose(maxFileUploadSize)
		if !p.upload {
			continue
		}
		if p.size <= budget {
			budget -= p.size
			continue
		}
		p.chooseURL()
	}

	downloadUploads(prepared)
	return prepared
}

func downloadUploads(prepared []*preparedMedium) {
	var wg sync.WaitGroup
	for _, p := range prepared {
		if p.upload && p.data == nil {
			wg.Go(func() {
				if err := p.download(); err != nil {
					log.Err(err).Str("url", p.url).Msg("Error while downloading")
					p.chooseURL()
				}
			})
		}
	}
	wg.Wait()
}

func retryMedia(prepared []*preparedMedium) {
	budget := int64(maxRequestUploadSize)
	for _, p := range prepared {
		if p.upload {
			budget -= p.size
		}
	}

	for _, p := range prepared {
		if p.url == "" || p.upload {
			continue
		}
		if p.size > 0 && p.size <= budget {
			p.upload = true
			budget -= p.size
			continue
		}
		p.url = bustCache(p.url)
	}

	downloadUploads(prepared)
}

func bustCache(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return link
	}
	q := u.Query()
	q.Set("gobot", xid.New().String())
	u.RawQuery = q.Encode()
	return u.String()
}

func (m *Medium) variantURLs() []string {
	if m.IsPhoto() {
		return []string{m.MediaUrlHttps + ":orig", m.MediaUrlHttps + "?name=4096x4096", m.MediaUrlHttps + "?name=large"}
	}

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

const (
	sizeUnknown int64 = -1
	sizeMissing int64 = -2
)

func headSize(link string) int64 {
	resp, err := httpUtils.DefaultHttpClient.Head(link)
	if err != nil {
		return sizeUnknown
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return sizeMissing
	}
	if resp.StatusCode != http.StatusOK {
		return sizeUnknown
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

func sendAlbums(b *gotgbot.Bot, c plugin.GobotContext, prepared []*preparedMedium, spoiler bool) {
	var failed []*preparedMedium
	usable := make([]*preparedMedium, 0, len(prepared))
	for _, p := range prepared {
		if p.usable() {
			usable = append(usable, p)
		} else {
			failed = append(failed, p)
		}
	}

	if len(usable) > 0 {
		albumCount := (len(usable) + tgUtils.MaxAlbumSize - 1) / tgUtils.MaxAlbumSize
		albumSize := (len(usable) + albumCount - 1) / albumCount
		for chunk := range slices.Chunk(usable, albumSize) {
			if err := replyAlbum(b, c, chunk, spoiler); err != nil {
				log.Err(err).Msg("Error while sending album")
				failed = append(failed, chunk...)
			}
		}
	}

	if len(failed) == 0 {
		return
	}
	slices.SortFunc(failed, func(a, b *preparedMedium) int { return a.position - b.position })
	links := make([]string, len(failed))
	for i, p := range failed {
		links[i] = mediumLink(p, cmpLink(p), len(prepared))
	}
	_, err := c.EffectiveMessage.ReplyMessage(b, "🔗 "+strings.Join(links, " · "), &gotgbot.SendMessageOpts{
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
	})
	if err != nil {
		log.Err(err).Msg("Error while sending media links")
	}
}

func replyAlbum(b *gotgbot.Bot, c plugin.GobotContext, media []*preparedMedium, spoiler bool) error {
	action := gotgbot.ChatActionUploadPhoto
	album := make([]gotgbot.InputMedia, 0, len(media))
	for i, p := range media {
		file := p.inputFile(fmt.Sprintf("file%d", i))
		if p.medium.IsPhoto() {
			album = append(album, gotgbot.InputMediaPhoto{Media: file, HasSpoiler: spoiler})
		} else {
			action = gotgbot.ChatActionUploadVideo
			album = append(album, gotgbot.InputMediaVideo{Media: file, SupportsStreaming: true, HasSpoiler: spoiler})
		}
	}
	_, _ = c.EffectiveChat.SendAction(b, action, nil)

	if len(album) == 1 {
		return replySingle(b, c, album[0])
	}

	_, err := c.EffectiveMessage.ReplyMediaGroup(b, album, &gotgbot.SendMediaGroupOpts{
		DisableNotification: true,
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		RequestOpts:         &gotgbot.RequestOpts{Timeout: uploadTimeout},
	})
	return err
}

func replySingle(b *gotgbot.Bot, c plugin.GobotContext, medium gotgbot.InputMedia) error {
	reply := &gotgbot.ReplyParameters{AllowSendingWithoutReply: true}
	timeout := &gotgbot.RequestOpts{Timeout: uploadTimeout}
	switch m := medium.(type) {
	case gotgbot.InputMediaPhoto:
		_, err := c.EffectiveMessage.ReplyPhoto(b, m.Media, &gotgbot.SendPhotoOpts{
			ReplyParameters:     reply,
			DisableNotification: true,
			HasSpoiler:          m.HasSpoiler,
			RequestOpts:         timeout,
		})
		return err
	case gotgbot.InputMediaVideo:
		_, err := c.EffectiveMessage.ReplyVideo(b, m.Media, &gotgbot.SendVideoOpts{
			ReplyParameters:     reply,
			DisableNotification: true,
			SupportsStreaming:   true,
			HasSpoiler:          m.HasSpoiler,
			RequestOpts:         timeout,
		})
		return err
	}
	return errors.New("unsupported media type")
}

func (p *preparedMedium) inputFile(name string) gotgbot.InputFileOrString {
	if p.upload && p.data != nil {
		return gotgbot.InputFileByReader(name, bytes.NewReader(p.data))
	}
	return gotgbot.InputFileByURL(p.url)
}

func cmpLink(p *preparedMedium) string {
	if p.fullURL != "" {
		return p.fullURL
	}
	if p.url != "" {
		return p.url
	}
	return p.medium.Link()
}
