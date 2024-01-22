package utils

import (
	"errors"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
)

var (
	DefaultSendOptions = &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	}
	log = logger.New("utils")
)

type (
	VersionInfo struct {
		GoVersion  string
		GoOS       string
		GoArch     string
		Revision   string
		LastCommit time.Time
		DirtyBuild bool
	}
)

func ReadVersionInfo() (VersionInfo, error) {
	buildInfo, ok := debug.ReadBuildInfo()

	if !ok {
		return VersionInfo{}, errors.New("could not read build info")
	}

	versionInfo := VersionInfo{
		GoVersion: buildInfo.GoVersion,
	}

	for _, kv := range buildInfo.Settings {
		switch kv.Key {
		case "GOOS":
			versionInfo.GoOS = kv.Value
		case "GOARCH":
			versionInfo.GoArch = kv.Value
		case "vcs.revision":
			versionInfo.Revision = kv.Value
		case "vcs.time":
			versionInfo.LastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			versionInfo.DirtyBuild = kv.Value == "true"
		}
	}

	return versionInfo, nil
}

func GermanTimezone() *time.Location {
	timezone, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Err(err).Msg("Failed to load timezone, using UTC")
		timezone, _ = time.LoadLocation("UTC")
	}
	return timezone
}

func AnyEntities(message *gotgbot.Message) []gotgbot.MessageEntity {
	entities := message.Entities
	if message.Entities == nil {
		entities = message.CaptionEntities
	}
	return entities
}

func AnyText(message *gotgbot.Message) string {
	text := message.Text
	if message.Text == "" {
		text = message.Caption
	}
	return text
}

func ContainsMedia(m *gotgbot.Message) bool {
	switch {
	case m.Photo != nil:
		return true
	case m.Voice != nil:
		return true
	case m.Audio != nil:
		return true
	case m.Animation != nil:
		return true
	case m.Sticker != nil:
		return true
	case m.Document != nil:
		return true
	case m.Video != nil:
		return true
	case m.VideoNote != nil:
		return true
	default:
		return false
	}
}

func TimestampToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

func GetBestResolution(photo []gotgbot.PhotoSize) *gotgbot.PhotoSize {
	if photo == nil {
		return nil
	}
	var filesize int64
	var bestResolution *gotgbot.PhotoSize
	for _, photoSize := range photo {
		photoSize := photoSize
		if photoSize.FileSize > filesize {
			filesize = photoSize.FileSize
			bestResolution = &photoSize
		}
	}

	return bestResolution
}

func IsAdmin(user *gotgbot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.Id
}

func FromGroup(message gotgbot.MaybeInaccessibleMessage) bool {
	return message.GetChat().Type == gotgbot.ChatTypeGroup || message.GetChat().Type == gotgbot.ChatTypeSupergroup
}

func IsPrivate(message *gotgbot.Message) bool {
	return message.Chat.Type == gotgbot.ChatTypePrivate
}
