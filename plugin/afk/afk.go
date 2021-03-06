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
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("afk")

type (
	Plugin struct {
		afkService Service
	}

	Service interface {
		BackAgain(chat *telebot.Chat, user *telebot.User) error
		IsAFK(chat *telebot.Chat, user *telebot.User) (bool, model.AFKData, error)
		IsAFKByUsername(chat *telebot.Chat, username string) (bool, model.AFKData, error)
		SetAFK(chat *telebot.Chat, user *telebot.User, now time.Time) error
		SetAFKWithReason(chat *telebot.Chat, user *telebot.User, reason string) error
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "afk",
			Description: "[Text] - Auf AFK schalten",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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
			Trigger:     telebot.EntityMention,
			HandlerFunc: p.notifyIfAFK,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) goAFK(c plugin.GobotContext) error {
	var reason string
	if len(c.Matches) > 1 {
		reason = c.Matches[1]
	}
	var err error

	if reason != "" {
		err = p.afkService.SetAFKWithReason(c.Chat(), c.Sender(), reason)
	} else {
		err = p.afkService.SetAFK(c.Chat(), c.Sender(), time.Now())
	}

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("reason", reason).
			Msg("Failure to go AFK")
		return c.Reply(fmt.Sprintf("??? Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"???? <b>%s ist jetzt AFK</b>",
			utils.Escape(c.Sender().FirstName),
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

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func (p *Plugin) checkAFK(c plugin.GobotContext) error {
	if strings.HasPrefix(c.Text(), "/afk") {
		return nil
	}

	isAFK, data, err := p.afkService.IsAFK(c.Chat(), c.Sender())
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Msg("Failure to check AFK")
		return nil
	}

	if !isAFK {
		return nil
	}

	err = p.afkService.BackAgain(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Msg("Failure to set back again")
		return c.Reply(fmt.Sprintf("??? Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"???? <b>%s ist wieder da!</b> <i>(???? %s",
			utils.Escape(c.Sender().FirstName),
			data.Duration().Round(time.Second),
		),
	)

	if data.Reason.Valid {
		sb.WriteString(
			fmt.Sprintf(
				", ???? %s",
				utils.Escape(data.Reason.String),
			),
		)
	}
	sb.WriteString(")</i>")

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func (p *Plugin) notifyIfAFK(c plugin.GobotContext) error {
	var mentionedUsername string
	for _, entity := range utils.AnyEntities(c.Message()) {
		if entity.Type == telebot.EntityMention {
			if mentionedUsername != "" {
				return nil // Supports only one username
			}
			username := strings.TrimPrefix(c.Message().EntityText(entity), "@")
			mentionedUsername = strings.ToLower(username)
		}
	}

	if mentionedUsername == "" ||
		mentionedUsername == strings.ToLower(c.Bot().Me.Username) ||
		mentionedUsername == strings.ToLower(c.Sender().Username) {
		return nil
	}

	isAFK, data, err := p.afkService.IsAFKByUsername(c.Chat(), mentionedUsername)
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
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
			"???? <b>%s ist zurzeit AFK!</b> <i>(???? seit %s",
			utils.Escape(data.FirstName),
			data.Duration().Round(time.Second),
		),
	)

	if data.Reason.Valid {
		sb.WriteString(
			fmt.Sprintf(
				", ???? %s",
				utils.Escape(data.Reason.String),
			),
		)
	}

	sb.WriteString(")</i>")

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
