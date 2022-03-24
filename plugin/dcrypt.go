package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
	"io"
	"log"
	"regexp"
	"strings"
)

type (
	DcryptPlugin struct {
		*bot.Plugin
		textRegex *regexp.Regexp
	}

	DcryptItResponse struct {
		FormErrors struct {
			Dlcfile []string `json:"dlcfile"`
		} `json:"form_errors"`
		Success struct {
			Message string   `json:"message"`
			Links   []string `json:"links"`
		} `json:"success"`
	}
)

func (plg *DcryptPlugin) Init() {
	plg.textRegex = regexp.MustCompile("(?s)<textarea>(.+)</textarea>")
}

func (*DcryptPlugin) GetName() string {
	return "dcrypt"
}

func (plg *DcryptPlugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: telebot.OnDocument,
			Handler: plg.OnFile,
		},
	}
}

func (plg *DcryptPlugin) OnFile(c bot.NextbotContext) error {
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
		log.Println(err)
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
		log.Println(err)
		return c.Reply("❌ Konnte Datei nicht zu dcrypt.it hochladen.", utils.DefaultSendOptions)
	}

	if resp.StatusCode == 413 {
		return c.Reply("❌ Container ist zum Entschlüsseln zu groß.", utils.DefaultSendOptions)
	}

	if resp.StatusCode != 200 {
		return c.Reply(fmt.Sprintf(
			"❌ dcrypt.it konnte nicht erreicht werden: HTTP-Fehler %d",
			resp.StatusCode,
		),
			utils.DefaultSendOptions)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	matches := plg.textRegex.FindStringSubmatch(string(body))
	if matches == nil {
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	var data DcryptItResponse
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		log.Println(err)
		return c.Reply("❌ Konnte Antwort von dcrypt.it nicht lesen.", utils.DefaultSendOptions)
	}

	if data.Success.Message == "" {
		log.Println(data.FormErrors.Dlcfile)
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
