package plugin

import (
	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
	"regexp"
)

// TODO: Bot muss an Plugin rangehängt werden weil man DB im Init() braucht (keys laden!)
type AboutPlugin struct {
	*bot.Plugin
	key string
}

// oder bot hier als parameter? aber unten dann auch hmm
func (plg *AboutPlugin) Init() {
	plg.Plugin = bot.NewPlugin("about", []bot.Handler{
		{
			Command: regexp.MustCompile(`^/about$`),
			Handler: plg.OnAbout,
		},
	})
	plg.key = "geheim"
}

func (plg *AboutPlugin) OnAbout(c telebot.Context) error {
	// TOOD: Context um Matches erweitern o.ä.?
	return c.Send(plg.key)
}
