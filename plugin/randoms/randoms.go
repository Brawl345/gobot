package randoms

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
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

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "random",
			Description: "<Nutzer> - Schabernack",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func (p *Plugin) addRandom(b *gotgbot.Bot, c plugin.GobotContext) error {
	random := c.Matches[1]

	if !strings.Contains(random, "{user}") ||
		!strings.Contains(random, "{other_user}") {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Dein Text muss <code>{user}</code> und <code>{other_user}</code> enthalten, welche durch die Usernamen ersetzt werden.", utils.DefaultSendOptions())
		return err
	}

	err := p.randomService.SaveRandom(random)

	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			_, err := c.EffectiveMessage.ReplyMessage(b, "<b>💡 Text existiert bereits!</b>", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("random", random).
			Msg("failed to save random")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	example := strings.NewReplacer(
		"{user}", "<b>"+utils.Escape(c.EffectiveUser.FirstName)+"</b>",
		"{other_user}", "<b>"+utils.Escape(b.FirstName)+"</b>",
	).Replace(utils.Escape(random))

	_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("<b>✅ Gespeichert!</b> Beispiel:\n%s", example),
		utils.DefaultSendOptions())
	return err
}

func (p *Plugin) delRandom(b *gotgbot.Bot, c plugin.GobotContext) error {
	random := c.Matches[1]
	err := p.randomService.DeleteRandom(random)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err := c.EffectiveMessage.ReplyMessage(b, "<b>❌ Nicht gefunden!</b>", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("random", random).
			Msg("failed to delete random")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "<b>✅ Text gelöscht!</b>",
	})
}

func (p *Plugin) random(b *gotgbot.Bot, c plugin.GobotContext) error {
	random, err := p.randomService.GetRandom()
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err := c.EffectiveMessage.ReplyMessage(b, "<b>❌ Keine Texte gefunden!</b> Bitte doch den Bot-Administrator darum, welche einzuspeichern.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("failed to get random")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	random = strings.NewReplacer(
		"{user}", "<b>"+utils.Escape(c.EffectiveUser.FirstName)+"</b>",
		"{other_user}", "<b>"+utils.Escape(c.Matches[1])+"</b>",
	).Replace(random)
	_, err = c.EffectiveMessage.ReplyMessage(b, random, utils.DefaultSendOptions())
	return err
}
