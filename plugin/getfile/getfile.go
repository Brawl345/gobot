package getfile

import (
	"cmp"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils/httpUtils"
	tgUtils "github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var log = logger.New("getfile")

type (
	Plugin struct {
		credentialService model.CredentialService
		fileService       Service
	}

	Service interface {
		Create(uniqueID, fileName, mediaType string) error
		Exists(uniqueID string) (bool, error)
	}
)

func New(credentialService model.CredentialService, fileService Service) *Plugin {
	return &Plugin{
		fileService:       fileService,
		credentialService: credentialService,
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
			Trigger:     tgUtils.AnyMedia,
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

	if c.EffectiveMessage.Animation != nil {
		fileID = c.EffectiveMessage.Animation.FileId
		fileSize = c.EffectiveMessage.Animation.FileSize
		uniqueID = c.EffectiveMessage.Animation.FileUniqueId
		subFolder = "animation"
	} else if c.EffectiveMessage.Audio != nil {
		fileID = c.EffectiveMessage.Audio.FileId
		fileSize = c.EffectiveMessage.Audio.FileSize
		uniqueID = c.EffectiveMessage.Audio.FileUniqueId
		subFolder = "audio"
	} else if c.EffectiveMessage.Document != nil {
		fileID = c.EffectiveMessage.Document.FileId
		fileSize = c.EffectiveMessage.Document.FileSize
		uniqueID = c.EffectiveMessage.Document.FileUniqueId
		subFolder = "document"
	} else if c.EffectiveMessage.Photo != nil {
		bestResolution := tgUtils.GetBestResolution(c.EffectiveMessage.Photo)
		fileID = bestResolution.FileId
		fileSize = bestResolution.FileSize
		uniqueID = bestResolution.FileUniqueId
		subFolder = "photo"
	} else if c.EffectiveMessage.Sticker != nil {
		fileID = c.EffectiveMessage.Sticker.FileId
		fileSize = c.EffectiveMessage.Sticker.FileSize
		uniqueID = c.EffectiveMessage.Sticker.FileUniqueId
		subFolder = "sticker"
	} else if c.EffectiveMessage.Video != nil {
		fileID = c.EffectiveMessage.Video.FileId
		fileSize = c.EffectiveMessage.Video.FileSize
		uniqueID = c.EffectiveMessage.Video.FileUniqueId
		subFolder = "video"
	} else if c.EffectiveMessage.VideoNote != nil {
		fileID = c.EffectiveMessage.VideoNote.FileId
		fileSize = c.EffectiveMessage.VideoNote.FileSize
		uniqueID = c.EffectiveMessage.VideoNote.FileUniqueId
		subFolder = "videoNote"
	} else if c.EffectiveMessage.Voice != nil {
		fileID = c.EffectiveMessage.Voice.FileId
		fileSize = c.EffectiveMessage.Voice.FileSize
		uniqueID = c.EffectiveMessage.Voice.FileUniqueId
		subFolder = "voice"
	}

	if fileSize > tgUtils.MaxFilesizeDownload {
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

	dir := cmp.Or(p.credentialService.GetKey("getfile_dir"), "files")

	savePath := filepath.Join(dir, subFolder)
	err = os.MkdirAll(savePath, 0770)
	if err != nil {
		log.Err(err).Msgf("Could not create directory: %s", savePath)
		return nil
	}

	file, err := b.GetFile(fileID, nil)
	if err != nil {
		log.Err(err).
			Str("fileID", fileID).
			Str("uniqueID", uniqueID).
			Str("mediaType", subFolder).
			Msg("Failed to get file from Telegram")
		return err
	}

	reader, err := httpUtils.DownloadFileFromGetFile(b, file)
	if err != nil {
		log.Err(err).
			Str("fileID", fileID).
			Str("uniqueID", uniqueID).
			Str("mediaType", subFolder).
			Msg("Failed to download file")
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
		!c.EffectiveMessage.Sticker.IsAnimated {
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
