package dcrypt

import (
	"io"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
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
			Trigger:     tgUtils.DocumentMsg,
			HandlerFunc: p.OnFile,
		},
	}
}

func (p *Plugin) OnFile(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.EffectiveMessage.Document.MimeType != "text/plain" ||
		!strings.HasSuffix(c.EffectiveMessage.Document.FileName, ".dlc") {
		return nil
	}

	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadDocument, nil)

	if c.EffectiveMessage.Document.FileSize > tgUtils.MaxFilesizeDownload {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ DLC-Container ist größer als 20 MB.", utils.DefaultSendOptions())
		return err
	}

	file, err := httpUtils.DownloadFile(b, c.EffectiveMessage.Document.FileId)
	if err != nil {
		log.Err(err).
			Interface("file", c.EffectiveMessage.Document).
			Msg("Failed to download file")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Konnte Datei nicht von Telegram herunterladen.", utils.DefaultSendOptions())
		return err
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Err(err).
			Interface("file", c.EffectiveMessage.Document).
			Msg("Failed to read file")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Konnte Datei nicht lesen.", utils.DefaultSendOptions())
		return err
	}

	dlc, err := DecryptDLC(fileData)

	if err != nil {
		log.Err(err).
			Interface("file", c.EffectiveMessage.Document).
			Msg("Failed to decrypt file")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Konnte DLC-Container nicht entschlüsseln.", utils.DefaultSendOptions())
		return err
	}

	if !dlc.HasLinks() {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Keine Links gefunden.", utils.DefaultSendOptions())
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

	for _, pkg := range dlc.Content.Package {
		for _, file := range pkg.File {
			sb.WriteString(string(file.URL))
			sb.WriteString("\n")
		}
	}

	document := gotgbot.InputFileByReader(filename, strings.NewReader(sb.String()))

	var sbCaption strings.Builder
	sbCaption.WriteString("🔑 Links entschlüsselt")

	size := dlc.TotalSize()
	if size != "" {
		sbCaption.WriteString("!\n")
		sbCaption.WriteString(size)
	}

	generatedBy := dlc.GeneratedBy()
	if generatedBy != "" {
		if size == "" {
			sbCaption.WriteString("!")
		}
		sbCaption.WriteString("\n")
		sbCaption.WriteString(generatedBy)
	}

	if generatedBy == "" && size == "" {
		sbCaption.WriteString(".")
	}

	_, err = c.EffectiveMessage.ReplyDocument(b, document, &gotgbot.SendDocumentOpts{
		Caption: sbCaption.String(),
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	})
	return err
}
