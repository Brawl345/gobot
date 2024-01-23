package getfile

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
)

var log = logger.New("getfile")

type (
	Plugin struct {
		fileService Service
		dir         string
	}

	Service interface {
		Create(uniqueID, fileName, mediaType string) error
		Exists(uniqueID string) (bool, error)
	}
)

func New(credentialService model.CredentialService, fileService Service) *Plugin {
	dir, err := credentialService.GetKey("getfile_dir")
	if err != nil {
		dir = "files"
	}
	return &Plugin{
		fileService: fileService,
		dir:         dir,
	}
}

func (*Plugin) Name() string {
	return "getfile"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     telebot.OnMedia,
			HandlerFunc: p.OnMedia,
			HandleEdits: true,
		},
	}
}

func (p *Plugin) OnMedia(b *gotgbot.Bot, c plugin.GobotContext) error {
	var fileID string
	var uniqueID string
	var subFolder string
	var fileSize int64

	if c.EffectiveMessage.Media() != nil {
		fileID = c.EffectiveMessage.Media().MediaFile().FileID
		fileSize = c.EffectiveMessage.Media().MediaFile().FileSize
		uniqueID = c.EffectiveMessage.Media().MediaFile().UniqueID
		subFolder = c.EffectiveMessage.Media().MediaType()
	} else {
		fileID = c.EffectiveMessage.Sticker.FileID
		fileSize = c.EffectiveMessage.Sticker.FileSize
		uniqueID = c.EffectiveMessage.Sticker.UniqueID
		subFolder = c.EffectiveMessage.Sticker.MediaType()
	}

	if fileSize > utils.MaxFilesizeDownload {
		log.Warn().Msgf("File is too big: %d", fileSize)
		return nil
	}

	exists, err := p.fileService.Exists(uniqueID)
	if err != nil {
		log.Err(err).Msg("Error checking if file exists")
		return nil
	}

	if exists {
		log.Info().Msgf("File already exists: %s", uniqueID)
		return nil
	}

	savePath := filepath.Join(p.dir, subFolder)
	err = os.MkdirAll(savePath, 0770)
	if err != nil {
		log.Err(err).Msgf("Could not create directory: %s", savePath)
		return nil
	}

	file := &telebot.File{FileID: fileID}
	reader, err := c.Bot().File(file)
	if err != nil {
		return err
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			log.Err(err).Msg("Error closing file")
		}
	}(reader)

	var fileName string
	if c.EffectiveMessage.Animation != nil && c.EffectiveMessage.Animation.FileName != "" {
		fileName = uniqueID + "_" + c.EffectiveMessage.Animation.FileName
	} else if c.EffectiveMessage.Audio != nil && c.EffectiveMessage.Audio.FileName != "" {
		fileName = uniqueID + "_" + c.EffectiveMessage.Audio.FileName
	} else if c.EffectiveMessage.Document != nil && c.EffectiveMessage.Document.FileName != "" {
		fileName = uniqueID + "_" + c.EffectiveMessage.Document.FileName
	} else if c.EffectiveMessage.Video != nil && c.EffectiveMessage.Video.FileName != "" {
		fileName = uniqueID + "_" + c.EffectiveMessage.Video.FileName
	} else {
		fileName = path.Base(file.FilePath)
	}

	// Fix file endings
	if c.EffectiveMessage.Sticker != nil &&
		!c.EffectiveMessage.Sticker.Animated {
		if !strings.HasSuffix(fileName, ".webp") && !strings.HasSuffix(fileName, ".webm") {
			fileName += ".webp"
		}
	}
	if c.EffectiveMessage.Voice != nil &&
		!strings.HasSuffix(fileName, ".oga") {
		fileName += ".oga"
	}
	if c.EffectiveMessage.VideoNote != nil &&
		!strings.HasSuffix(fileName, ".mp4") {
		fileName += ".mp4"
	}

	out, err := os.Create(filepath.Join(savePath, fileName))
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Err(err).Msg("Error closing file")
		}
	}(out)

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}
	log.Info().Msgf("Saved as: %s", filepath.Join(savePath, fileName))

	err = p.fileService.Create(uniqueID, fileName, subFolder)
	if err != nil {
		return err
	}

	return nil
}
