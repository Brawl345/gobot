package speech_to_text

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var log = logger.New("speech_to_text")

const (
	ApiUrl       = "https://api.openai.com/v1/audio/transcriptions"
	MaxVoiceSize = 25000000 // File uploads to Whisper are limited to 25 MB
	MaxDuration  = 180      // 3 minutes
)

type (
	Plugin struct {
		credentialService model.CredentialService
	}
)

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (p *Plugin) Name() string {
	return "speech_to_text"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     tgUtils.VoiceMsg,
			HandlerFunc: p.OnVoice,
		},
	}
}

func (p *Plugin) OnVoice(b *gotgbot.Bot, c plugin.GobotContext) error {
	apiKey := p.credentialService.GetKey("openai_api_key")
	if apiKey == "" {
		log.Warn().Msg("openai_api_key not found")
		return nil
	}

	if c.EffectiveMessage.Voice.FileSize > tgUtils.MaxFilesizeDownload {
		log.Warn().
			Int64("filesize", c.EffectiveMessage.Voice.FileSize).
			Msg("Voice file is too big to download")
		return nil
	}

	if c.EffectiveMessage.Voice.FileSize > MaxVoiceSize {
		log.Warn().
			Int64("filesize", c.EffectiveMessage.Voice.FileSize).
			Msg(fmt.Sprintf("Voice file is bigger than %d bytes", MaxVoiceSize))
		return nil
	}

	if c.EffectiveMessage.Voice.Duration > MaxDuration {
		log.Warn().
			Int64("duration", c.EffectiveMessage.Voice.Duration).
			Msg(fmt.Sprintf("Voice message is longer than %d seconds", MaxDuration))
		return nil
	}

	file, err := httpUtils.DownloadFile(b, c.EffectiveMessage.Voice.FileId)
	if err != nil {
		log.Err(err).
			Interface("file", c.EffectiveMessage.Voice).
			Msg("Failed to download file")
		return nil
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	fileEnding := ".ogg"
	if c.EffectiveMessage.Voice.MimeType == "audio/mpeg" {
		fileEnding = ".mp3"
	} else if c.EffectiveMessage.Voice.MimeType == "audio/mp4" {
		fileEnding = ".m4a"
	}

	resp, err := httpUtils.MultiPartFormRequestWithHeaders(
		ApiUrl,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		},
		[]httpUtils.MultiPartParam{
			{
				Name:  "model",
				Value: "whisper-1",
			},
		},
		[]httpUtils.MultiPartFile{
			{
				FieldName: "file",
				FileName:  fmt.Sprintf("voice%s", fileEnding),
				Content:   file,
			},
		},
	)

	if err != nil {
		log.Err(err).Msg("Failed to upload file")
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	if err != nil {
		log.Err(err).Msg("Failed to read body")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ApiErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			log.Error().
				Int("status_code", resp.StatusCode).
				Msg("Failed to upload file")
			return nil
		}

		log.Error().
			Int("status_code", resp.StatusCode).
			Str("error_message", errorResponse.Error.Message).
			Str("error_type", errorResponse.Error.Type).
			Msg("Failed to upload file")
		return nil
	}

	var apiResponse ApiResponse

	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		log.Err(err).Msg("Failed to parse body")
		return nil
	}

	if len(apiResponse.Text) == 0 {
		log.Warn().Msg("Voice message contains no text")
		return nil
	}

	var sb strings.Builder

	sb.WriteString("💬 ")
	if len(apiResponse.Text) > tgUtils.MaxMessageLength {
		sb.WriteString(apiResponse.Text[:tgUtils.MaxMessageLength-10])
	} else {
		sb.WriteString(apiResponse.Text)
	}

	_, err = c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ReplyParameters:     &gotgbot.ReplyParameters{AllowSendingWithoutReply: true},
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
		DisableNotification: true,
	})
	return err
}
