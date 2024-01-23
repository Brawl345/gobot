package dcrypt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("dcrypt")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "dcrypt"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     utils.DocumentMsg,
			HandlerFunc: p.OnFile,
		},
	}
}

func (p *Plugin) OnFile(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.EffectiveMessage.Document.MimeType != "text/plain" ||
		!strings.HasSuffix(c.EffectiveMessage.Document.FileName, ".dlc") {
		return nil
	}

	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionUploadDocument, nil)

	if c.EffectiveMessage.Document.FileSize > utils.MaxFilesizeDownload {
		_, err := c.EffectiveMessage.Reply(b, "❌ DLC-Container ist größer als 20 MB.", utils.DefaultSendOptions())
		return err
	}

	file, err := httpUtils.DownloadFile(b, c.EffectiveMessage.Document.FileId)
	if err != nil {
		log.Err(err).
			Interface("file", c.EffectiveMessage.Document).
			Msg("Failed to download file")
		_, err := c.EffectiveMessage.Reply(b, "❌ Konnte Datei nicht von Telegram herunterladen.", utils.DefaultSendOptions())
		return err
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	resp, err := httpUtils.MultiPartFormRequest(
		"https://dcrypt.it/decrypt/upload",
		[]httpUtils.MultiPartParam{},
		[]httpUtils.MultiPartFile{
			{
				FieldName: "dlcfile",
				FileName:  "dlc.dlc",
				Content:   file,
			},
		},
	)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to upload file")
		_, err := c.EffectiveMessage.Reply(b,
			fmt.Sprintf("❌ Konnte Datei nicht zu dcrypt.it hochladen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions(),
		)
		return err
	}

	if resp.StatusCode == 413 {
		log.Error().Msg("File is too big")
		_, err := c.EffectiveMessage.Reply(b, "❌ Container ist zum Entschlüsseln zu groß.", utils.DefaultSendOptions())
		return err
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status_code", resp.StatusCode).Msg("Failed to upload file")
		_, err := c.EffectiveMessage.Reply(b,
			fmt.Sprintf(
				"❌ dcrypt.it konnte nicht erreicht werden: HTTP-Fehler %d",
				resp.StatusCode,
			),
			utils.DefaultSendOptions(),
		)
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to read response body")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Konnte Antwort von dcrypt.it nicht lesen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	matches := textRegex.FindStringSubmatch(string(body))
	if matches == nil {
		_, err := c.EffectiveMessage.Reply(b, "❌ dcrypt.it hat keine Links gefunden.", utils.DefaultSendOptions())
		return err
	}

	var data Response
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to unmarshal response body")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Konnte Antwort von dcrypt.it nicht lesen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if data.Success.Message == "" {
		guid := xid.New().String()
		log.
			Error().
			Str("guid", guid).
			Strs("form_errors", data.FormErrors.Dlcfile).
			Msg("Failed to decrypt DLC")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ DLC-Container konnte nicht gelesen werden.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	var filename string
	if c.EffectiveMessage.Document.FileName == "" {
		filename = "Links.txt"
	} else {
		filename = strings.TrimSuffix(c.EffectiveMessage.Document.FileName, ".dlc")
		filename = filename + ".txt"
	}

	var sb strings.Builder
	for _, link := range data.Success.Links {
		sb.WriteString(link)
		sb.WriteString("\n")
	}

	buf := bytes.NewBufferString(sb.String())
	document := &gotgbot.NamedFile{
		File:     buf,
		FileName: filename,
	}

	_, err = b.SendDocument(c.EffectiveChat.Id, document, &gotgbot.SendDocumentOpts{
		Caption: "🔑 Hier sind deine entschlüsselten Links!",
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	})
	return err
}
