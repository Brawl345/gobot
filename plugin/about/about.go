package about

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
)

var log = logger.NewLogger("about")

type Plugin struct {
	*bot.Plugin
	text string
}

func (*Plugin) GetName() string {
	return "about"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/about|start(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler: plg.OnAbout,
		},
	}
}

func (plg *Plugin) Init() {
	var (
		Revision   = "unknown"
		LastCommit time.Time
		DirtyBuild = true
	)
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		log.Error().Msg("Failed to read build info")
		return
	}

	for _, kv := range buildInfo.Settings {
		switch kv.Key {
		case "vcs.revision":
			Revision = kv.Value
		case "vcs.time":
			LastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			DirtyBuild = kv.Value == "true"
		}
	}

	text := fmt.Sprintf("<code>%s</code>\n<i>Comitted on %s</i>", Revision, LastCommit)
	if DirtyBuild {
		text += " (dirty)"
	}

	plg.text = text
}

func (plg *Plugin) OnAbout(c bot.NextbotContext) error {
	return c.Reply("Gobot "+plg.text, utils.DefaultSendOptions)
}
