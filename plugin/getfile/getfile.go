package getfile

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("getfile")

type Plugin struct {
	*bot.Plugin
	dir string
}

func New(base *bot.Plugin) *Plugin {
	dir, err := base.Bot.DB.Credentials.GetKey("getfile_dir")
	if err != nil {
		dir = "tmp"
	}
	return &Plugin{
		Plugin: base,
		dir:    dir,
	}
}

func (*Plugin) Name() string {
	return "getfile"
}

func (plg *Plugin) CommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
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

func (plg *Plugin) OnMedia(c bot.NextbotContext) error {
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

	if fileSize > utils.MaxFilesizeDownload {
		log.Warn().Msgf("File is too big: %d", fileSize)
		return nil
	}

	exists, _ := plg.Bot.DB.Files.Exists(uniqueID)

	if exists {
		log.Info().Msgf("File already exists: %s", uniqueID)
		return nil
	}

	savePath := filepath.Join(plg.dir, subFolder)
	err := os.MkdirAll(savePath, 0660)
	if err != nil {
		log.Error().Msgf("Could not create directory: %s", savePath)
		return nil
	}

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
	log.Info().Msgf("Saved as: %s", filepath.Join(savePath, fileName))

	err = plg.Bot.DB.Files.Create(uniqueID, fileName, subFolder)
	if err != nil {
		return err
	}

	return nil
}
