package dcrypt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
	"io"
	"regexp"
	"strings"
)

var log = logger.NewLogger("dcrypt")

type (
	Plugin struct {
		*bot.Plugin
		textRegex *regexp.Regexp
	}

	Response struct {
		FormErrors struct {
			Dlcfile []string `json:"dlcfile"`
		} `json:"form_errors"`
		Success struct {
			Message string   `json:"message"`
			Links   []string `json:"links"`
		} `json:"success"`
	}
)

func (plg *Plugin) Init() {
	plg.textRegex = regexp.MustCompile("(?s)<textarea>(.+)</textarea>")
}

func (*Plugin) GetName() string {
	return "dcrypt"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: telebot.OnDocument,
			Handler: plg.OnFile,
		},
	}
}

func (plg *Plugin) OnFile(c bot.NextbotContext) error {
	if c.Message().Document.MIME != "text/plain" ||
		!strings.HasSuffix(c.Message().Document.FileName, ".dlc") {
		return nil
	}

	c.Notify(telebot.UploadingDocument)

	if c.Message().Document.FileSize > bot.MaxFilesizeDownload {
		return c.Reply("❌ DLC-Container ist größer als 20 MB.", utils.DefaultSendOptions)
	}

	file, err := plg.Bot.File(&telebot.File{FileID: c.Message().Document.FileID})
	if err != nil {
		log.Err(err).Msg("Failed to download file")
		return c.Reply("❌ Konnte Datei nicht von Telegram herunterladen.", utils.DefaultSendOptions)
	}
	defer file.Close()

	resp, err := bot.MultiPartFormRequest(
		"https://dcrypt.it/decrypt/upload",
		[]bot.MultiPartParam{},
		[]bot.MultiPartFile{
			{
				FieldName: "dlcfile",
				FileName:  "dlc.dlc",
				Content:   file,
			},
		},
	)
	if err != nil {
		log.Err(err).Msg("Failed to upload file")
		return c.Reply("❌ Konnte Datei nicht zu dcrypt.it hochladen.", utils.DefaultSendOptions)
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

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read response body")
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	matches := plg.textRegex.FindStringSubmatch(string(body))
	if matches == nil {
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	var data Response
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		log.Err(err).Msg("Failed to unmarshal response body")
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	if data.Success.Message == "" {
		log.
			Error().
			Strs("form_errors", data.FormErrors.Dlcfile).
			Msg("Failed to decrypt DLC")
		return c.Reply("❌ DLC-Container konnte nicht gelesen werden.", utils.DefaultSendOptions)
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
