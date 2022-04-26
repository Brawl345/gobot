package about

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

type Plugin struct {
	aboutText string
}

func New() *Plugin {
	versionInfo, err := utils.ReadVersionInfo()
	if err != nil {
		return &Plugin{
			aboutText: "Gobot",
		}
	}

	var sb strings.Builder

	sb.WriteString("<b>Gobot</b>")

	if versionInfo.Revision != "" {
		sb.WriteString(
			fmt.Sprintf(
				"\n<code>%s</code>",
				versionInfo.Revision,
			),
		)
	}

	if !versionInfo.LastCommit.IsZero() {
		sb.WriteString(
			fmt.Sprintf(
				"\n<i>Comitted: %s</i>",
				versionInfo.LastCommit,
			),
		)
		if versionInfo.DirtyBuild {
			sb.WriteString(" (dirty)")
		}
	}

	sb.WriteString(
		fmt.Sprintf(
			"\nKompiliert mit <code>%s</code> auf <code>%s</code>, <code>%s</code>",
			versionInfo.GoVersion,
			versionInfo.GoOS,
			versionInfo.GoArch,
		),
	)

	return &Plugin{
		aboutText: sb.String(),
	}
}

func (*Plugin) Name() string {
	return "about"
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/(?:about|start)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnAbout,
		},
	}
}

func (p *Plugin) OnAbout(c plugin.GobotContext) error {
	return c.Reply(p.aboutText, utils.DefaultSendOptions)
}
