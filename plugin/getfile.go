package plugin

import (
	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type GetFilePlugin struct {
	*bot.Plugin
}

func (*GetFilePlugin) GetName() string {
	return "getfile"
}

func (plg *GetFilePlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command:     telebot.OnMedia,
			Handler:     plg.OnMedia,
			HandleEdits: true,
		},
		{ // telebots Message.Media does not include Stickers :(
			Command:     telebot.OnSticker,
			Handler:     plg.OnMedia,
			HandleEdits: true,
		},
	}
}

func (plg *GetFilePlugin) OnMedia(c bot.NextbotContext) error {
	var fileID string
	var uniqueID string
	var subFolder string
	var fileSize int

	if c.Message().Media() != nil {
		fileID = c.Message().Media().MediaFile().FileID
		fileSize = c.Message().Media().MediaFile().FileSize
		uniqueID = c.Message().Media().MediaFile().UniqueID
		subFolder = c.Message().Media().MediaType()
	} else {
		fileID = c.Message().Sticker.FileID
		fileSize = c.Message().Sticker.FileSize
		uniqueID = c.Message().Sticker.UniqueID
		subFolder = c.Message().Sticker.MediaType()
	}

	if fileSize > bot.MaxFilesizeDownload {
		log.Println("File is too big to download")
		return nil
	}

	exists, _ := plg.Bot.DB.Files.Exists(uniqueID)

	if exists {
		log.Println("File was already downloaded")
		return nil
	}

	savePath := filepath.Join("tmp", subFolder)
	os.MkdirAll(savePath, 0660)

	file := &telebot.File{FileID: fileID}
	reader, err := plg.Bot.File(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	var fileName string
	if c.Message().Animation != nil && c.Message().Animation.FileName != "" {
		fileName = uniqueID + "_" + c.Message().Animation.FileName
	} else if c.Message().Audio != nil && c.Message().Audio.FileName != "" {
		fileName = uniqueID + "_" + c.Message().Audio.FileName
	} else if c.Message().Document != nil && c.Message().Document.FileName != "" {
		fileName = uniqueID + "_" + c.Message().Document.FileName
	} else if c.Message().Video != nil && c.Message().Video.FileName != "" {
		fileName = uniqueID + "_" + c.Message().Video.FileName
	} else {
		fileName = path.Base(file.FilePath)
	}

	// Fix file endings
	if c.Message().Sticker != nil &&
		!c.Message().Sticker.Animated &&
		!strings.HasSuffix(fileName, ".webp") {
		fileName += ".webp"
	}
	if c.Message().Voice != nil &&
		!strings.HasSuffix(fileName, ".oga") {
		fileName += ".oga"
	}
	if c.Message().VideoNote != nil &&
		!strings.HasSuffix(fileName, ".mp4") {
		fileName += ".mp4"
	}

	out, err := os.Create(filepath.Join(savePath, fileName))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}
	log.Println("Saved as", filepath.Join(savePath, fileName))

	err = plg.Bot.DB.Files.Create(uniqueID, fileName, subFolder)
	if err != nil {
		return err
	}

	return nil
}
