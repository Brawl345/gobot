package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"regexp"
	"runtime/debug"
	"time"
)

type AboutPlugin struct {
	*bot.Plugin
	text string
}

func (*AboutPlugin) GetName() string {
	return "about"
}

func (plg *AboutPlugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/about|start(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler: plg.OnAbout,
		},
	}
}

func (plg *AboutPlugin) Init() {
	var (
		Revision   = "unknown"
		LastCommit time.Time
		DirtyBuild = true
	)
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
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

func (plg *AboutPlugin) OnAbout(c bot.NextbotContext) error {
	return c.Reply("Gobot "+plg.text, utils.DefaultSendOptions)
}
