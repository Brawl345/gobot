package about

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

type Plugin struct {
	text string
}

func New() *Plugin {
	var (
		Revision   = "unknown"
		LastCommit time.Time
		DirtyBuild = true
	)
	buildInfo, ok := debug.ReadBuildInfo()

	text := "Gobot"
	if ok {
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

		text = fmt.Sprintf("<code>%s</code>\n<i>Comitted on %s</i>", Revision, LastCommit)
		if DirtyBuild {
			text += " (dirty)"
		}
	}

	return &Plugin{
		text: text,
	}
}

func (*Plugin) Name() string {
	return "about"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/about|start(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnAbout,
		},
	}
}

func (plg *Plugin) OnAbout(c plugin.NextbotContext) error {
	return c.Reply("Gobot "+plg.text, utils.DefaultSendOptions)
}
