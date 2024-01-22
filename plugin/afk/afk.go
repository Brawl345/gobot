package afk

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("afk")

type (
	Plugin struct {
		afkService Service
	}

	Service interface {
		BackAgain(chat *gotgbot.Chat, user *gotgbot.Sender) error
		IsAFK(chat *gotgbot.Chat, user *gotgbot.Sender) (bool, model.AFKData, error)
		IsAFKByUsername(chat *gotgbot.Chat, username string) (bool, model.AFKData, error)
		SetAFK(chat *gotgbot.Chat, user *gotgbot.Sender, now time.Time) error
		SetAFKWithReason(chat *gotgbot.Chat, user *gotgbot.Sender, reason string) error
	}
)

func New(afkService Service) *Plugin {
	return &Plugin{
		afkService: afkService,
	}
}

func (p *Plugin) Name() string {
	return "afk"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "afk",
			Description: "[Text] - Auf AFK schalten",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/afk(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.goAFK,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/afk(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.goAFK,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     utils.OnMsg,
			HandlerFunc: p.checkAFK,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     utils.EntityTypeMention,
			HandlerFunc: p.notifyIfAFK,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) goAFK(b *gotgbot.Bot, c plugin.GobotContext) error {
	var reason string
	if len(c.Matches) > 1 {
		reason = c.Matches[1]
	}
	var err error

	if reason != "" {
		err = p.afkService.SetAFKWithReason(c.EffectiveChat, c.EffectiveSender, reason)
	} else {
		err = p.afkService.SetAFK(c.EffectiveChat, c.EffectiveSender, time.Now())
	}

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveSender.Id()).
			Str("reason", reason).
			Msg("Failure to go AFK")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"💤 <b>%s ist jetzt AFK</b>",
			utils.Escape(c.EffectiveSender.FirstName()),
		),
	)

	if reason != "" {
		sb.WriteString(
			fmt.Sprintf(
				" <i>(%s)</i>",
				utils.Escape(reason),
			),
		)
	}

	sb.WriteString(".")

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}

func (p *Plugin) checkAFK(b *gotgbot.Bot, c plugin.GobotContext) error {
	if strings.HasPrefix(utils.AnyText(c.EffectiveMessage), "/afk") {
		return nil
	}

	isAFK, data, err := p.afkService.IsAFK(c.EffectiveChat, c.EffectiveSender)
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveSender.Id()).
			Msg("Failure to check AFK")
		return nil
	}

	if !isAFK {
		return nil
	}

	err = p.afkService.BackAgain(c.EffectiveChat, c.EffectiveSender)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveSender.Id()).
			Msg("Failure to set back again")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"🔔 <b>%s ist wieder da!</b> <i>(🕒 %s",
			utils.Escape(c.EffectiveSender.FirstName()),
			data.Duration().Round(time.Second),
		),
	)

	if data.Reason.Valid {
		sb.WriteString(
			fmt.Sprintf(
				", 💬 %s",
				utils.Escape(data.Reason.String),
			),
		)
	}
	sb.WriteString(")</i>")

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}

func (p *Plugin) notifyIfAFK(b *gotgbot.Bot, c plugin.GobotContext) error {
	var mentionedUsername string
	for _, entity := range utils.AnyEntities(c.EffectiveMessage) {
		if utils.EntityType(entity.Type) == utils.EntityTypeMention {
			if mentionedUsername != "" {
				return nil // Supports only one username
			}
			username := strings.TrimPrefix(c.EffectiveMessage.ParseEntity(entity).Text, "@")
			mentionedUsername = strings.ToLower(username)
		}
	}

	if mentionedUsername == "" ||
		mentionedUsername == strings.ToLower(b.Username) ||
		mentionedUsername == strings.ToLower(c.EffectiveSender.Username()) {
		return nil
	}

	isAFK, data, err := p.afkService.IsAFKByUsername(c.EffectiveChat, mentionedUsername)
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("username", mentionedUsername).
			Msg("Failure to check AFK")
		return nil
	}

	if !isAFK {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"⚠️ <b>%s ist zurzeit AFK!</b> <i>(🕒 seit %s",
			utils.Escape(data.FirstName),
			data.Duration().Round(time.Second),
		),
	)

	if data.Reason.Valid {
		sb.WriteString(
			fmt.Sprintf(
				", 💬 %s",
				utils.Escape(data.Reason.String),
			),
		)
	}

	sb.WriteString(")</i>")

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}
