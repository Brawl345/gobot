package utils

import (
	"errors"
	"runtime/debug"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
)

var log = logger.New("utils")

var germanTimezone = loadGermanTimezone()

func loadGermanTimezone() *time.Location {
	timezone, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Err(err).Msg("Failed to load timezone, using UTC")
		return time.UTC
	}
	return timezone
}

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

func DefaultSendOptions() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	}
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

func GermanTimezone() *time.Location {
	return germanTimezone
}

func TimestampToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

func Ptr[T any](v T) *T {
	return &v
}

func TruncateText(text string, maxLength int, suffix ...string) string {
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}
	result := string(runes[:maxLength])
	if len(suffix) > 0 {
		return result + suffix[0]
	}
	return result
}
