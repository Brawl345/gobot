package randoms

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("randoms")

type (
	Plugin struct {
		randomService Service
	}

	Service interface {
		DeleteRandom(random string) error
		GetRandom() (string, error)
		SaveRandom(random string) error
	}
)

func New(randomService Service) *Plugin {
	return &Plugin{
		randomService: randomService,
	}
}

func (p *Plugin) Name() string {
	return "randoms"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "random",
			Description: "<Nutzer> - Schabernack",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/addrandom(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.addRandom,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delrandom(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.delRandom,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/random(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.random,
		},
	}
}

func (p *Plugin) addRandom(c plugin.GobotContext) error {
	random := c.Matches[1]

	if !strings.Contains(random, "{user}") ||
		!strings.Contains(random, "{other_user}") {
		return c.Reply("❌ Dein Text muss <code>{user}</code> und <code>{other_user}</code> enthalten, welche durch die Usernamen ersetzt werden.", utils.DefaultSendOptions)
	}

	err := p.randomService.SaveRandom(random)

	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			return c.Reply("<b>💡 Text existiert bereits!</b>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("random", random).
			Msg("failed to save random")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	example := strings.NewReplacer(
		"{user}", c.Sender().FirstName,
		"{other_user}", c.Bot().Me.FirstName,
	).Replace(random)

	return c.Reply(fmt.Sprintf("<b>✅ Gespeichert!</b> Beispiel:\n%s", html.EscapeString(example)),
		utils.DefaultSendOptions)
}

func (p *Plugin) delRandom(c plugin.GobotContext) error {
	random := c.Matches[1]
	err := p.randomService.DeleteRandom(random)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.Reply("<b>❌ Nicht gefunden!</b>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("random", random).
			Msg("failed to delete random")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("<b>✅ Text gelöscht!</b>", utils.DefaultSendOptions)
}

func (p *Plugin) random(c plugin.GobotContext) error {
	random, err := p.randomService.GetRandom()
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.Reply("<b>❌ Keine Texte gefunden!</b> Bitte doch den Bot-Administrator darum, welche einzuspeichern.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("failed to get random")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	random = strings.NewReplacer(
		"{user}", c.Sender().FirstName,
		"{other_user}", c.Matches[1],
	).Replace(random)
	return c.Reply(random, &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
	})
}
