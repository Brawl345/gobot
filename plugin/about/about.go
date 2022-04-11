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

	sb.WriteString("<b>Gobot</b>\n")

	sb.WriteString(
		fmt.Sprintf(
			"<code>%s</code>\n",
			versionInfo.Revision,
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<i>Comitted: %s</i>",
			versionInfo.LastCommit,
		),
	)

	if versionInfo.DirtyBuild {
		sb.WriteString(" (dirty)")
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

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/about|start(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnAbout,
		},
	}
}

func (plg *Plugin) OnAbout(c plugin.GobotContext) error {
	return c.Reply(plg.aboutText, utils.DefaultSendOptions)
}
