package speech_to_text

import (
	"encoding/json"
	"fmt"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"gopkg.in/telebot.v3"
	"io"
	"strings"
)

var log = logger.New("speech_to_text")

const (
	ApiUrl       = "https://api.openai.com/v1/audio/transcriptions"
	MaxVoiceSize = 25000000 // "File uploads are currently limited to 25 MB"
	MaxDuration  = 180      // 3 minutes
)

type (
	Plugin struct {
		apiKey string
	}
)

func New(credentialService model.CredentialService) *Plugin {
	apiKey, err := credentialService.GetKey("openai_api_key")
	if err != nil {
		log.Warn().Msg("openai_api_key not found")
	}

	return &Plugin{
		apiKey: apiKey,
	}
}

func (p *Plugin) Name() string {
	return "speech_to_text"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     telebot.OnVoice,
			HandlerFunc: p.OnVoice,
		},
	}
}

func (p *Plugin) OnVoice(c plugin.GobotContext) error {
	if c.Message().Voice.FileSize > utils.MaxFilesizeDownload {
		log.Warn().
			Int64("filesize", c.Message().Voice.FileSize).
			Msg("Voice file is too big to download")
		return nil
	}

	if c.Message().Voice.FileSize > MaxVoiceSize {
		log.Warn().
			Int64("filesize", c.Message().Voice.FileSize).
			Msg(fmt.Sprintf("Voice file is bigger than %d bytes", MaxVoiceSize))
		return nil
	}

	if c.Message().Voice.Duration > MaxDuration {
		log.Warn().
			Int("duration", c.Message().Voice.Duration).
			Msg(fmt.Sprintf("Voice message is longer than %d seconds", MaxDuration))
		return nil
	}

	file, err := c.Bot().File(&telebot.File{FileID: c.Message().Voice.FileID})
	if err != nil {
		log.Err(err).
			Interface("file", c.Message().Voice).
			Msg("Failed to download file")
		return nil
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	resp, err := httpUtils.MultiPartFormRequestWithHeaders(
		ApiUrl,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", p.apiKey),
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
				FileName:  "voice.ogg",
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

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Err(err).Msg("Failed to read body")
		return nil
	}

	if resp.StatusCode != 200 {
		var errorResponse ApiErrorResponse
		if err := json.Unmarshal(body, &errorResponse); err != nil {
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

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Err(err).Msg("Failed to parse body")
		return nil
	}

	if len(apiResponse.Text) == 0 {
		log.Warn().Msg("Voice message contains no text")
		return nil
	}

	var sb strings.Builder

	sb.WriteString("💬 ")
	if len(apiResponse.Text) > utils.MaxMessageLength {
		sb.WriteString(apiResponse.Text[:utils.MaxMessageLength-10])
	} else {
		sb.WriteString(apiResponse.Text)
	}

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
	})
}