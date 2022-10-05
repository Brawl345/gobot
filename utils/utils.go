package utils

import (
	"errors"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/Brawl345/gobot/logger"
	"gopkg.in/telebot.v3"
)

var (
	DefaultSendOptions = &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
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

func GermanTimezone() *time.Location {
	timezone, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Err(err).Msg("Failed to load timezone, using UTC")
		timezone, _ = time.LoadLocation("UTC")
	}
	return timezone
}

func AnyEntities(message *telebot.Message) telebot.Entities {
	entities := message.Entities
	if message.Entities == nil {
		entities = message.CaptionEntities
	}
	return entities
}

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

func IsAdmin(user *telebot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.ID
}
