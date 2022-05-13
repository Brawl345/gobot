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
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("dcrypt")

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "dcrypt"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     telebot.OnDocument,
			HandlerFunc: p.OnFile,
		},
	}
}

func (p *Plugin) OnFile(c plugin.GobotContext) error {
	if c.Message().Document.MIME != "text/plain" ||
		!strings.HasSuffix(c.Message().Document.FileName, ".dlc") {
		return nil
	}

	_ = c.Notify(telebot.UploadingDocument)

	if c.Message().Document.FileSize > utils.MaxFilesizeDownload {
		return c.Reply("❌ DLC-Container ist größer als 20 MB.", utils.DefaultSendOptions)
	}

	file, err := c.Bot().File(&telebot.File{FileID: c.Message().Document.FileID})
	if err != nil {
		log.Err(err).
			Interface("file", c.Message().Document).
			Msg("Failed to download file")
		return c.Reply("❌ Konnte Datei nicht von Telegram herunterladen.", utils.DefaultSendOptions)
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	resp, err := utils.MultiPartFormRequest(
		"https://dcrypt.it/decrypt/upload",
		[]utils.MultiPartParam{},
		[]utils.MultiPartFile{
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
		return c.Reply(fmt.Sprintf("❌ Konnte Datei nicht zu dcrypt.it hochladen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if resp.StatusCode == 413 {
		log.Error().Msg("File is too big")
		return c.Reply("❌ Container ist zum Entschlüsseln zu groß.", utils.DefaultSendOptions)
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status_code", resp.StatusCode).Msg("Failed to upload file")
		return c.Reply(fmt.Sprintf(
			"❌ dcrypt.it konnte nicht erreicht werden: HTTP-Fehler %d",
			resp.StatusCode,
		),
			utils.DefaultSendOptions)
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
		return c.Reply(fmt.Sprintf("❌ Konnte Antwort von dcrypt.it nicht lesen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	matches := textRegex.FindStringSubmatch(string(body))
	if matches == nil {
		return c.Reply("❌ dcrypt.it hat keine Links gefunden.", utils.DefaultSendOptions)
	}

	var data Response
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to unmarshal response body")
		return c.Reply(fmt.Sprintf("❌ Konnte Antwort von dcrypt.it nicht lesen.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if data.Success.Message == "" {
		guid := xid.New().String()
		log.
			Error().
			Str("guid", guid).
			Strs("form_errors", data.FormErrors.Dlcfile).
			Msg("Failed to decrypt DLC")
		return c.Reply(fmt.Sprintf("❌ DLC-Container konnte nicht gelesen werden.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var filename string
	if c.Message().Document.FileName == "" {
		filename = "Links.txt"
	} else {
		filename = strings.TrimSuffix(c.Message().Document.FileName, ".dlc")
		filename = filename + ".txt"
	}

	var sb strings.Builder
	for _, link := range data.Success.Links {
		sb.WriteString(link)
		sb.WriteString("\n")
	}

	buf := bytes.NewBufferString(sb.String())

	document := &telebot.Document{
		File:     telebot.FromReader(buf),
		Caption:  "🔑 Hier sind deine entschlüsselten Links!",
		FileName: filename,
	}

	return c.Reply(document, utils.DefaultSendOptions)
}
