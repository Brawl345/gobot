package allow

import (
	"fmt"
	"html"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("allow")

type (
	Plugin struct {
		allowService Service
	}

	Service interface {
		AllowChat(chat *telebot.Chat) error
		AllowUser(user *telebot.User) error
		DenyChat(chat *telebot.Chat) error
		DenyUser(user *telebot.User) error
		IsChatAllowed(chat *telebot.Chat) bool
		IsUserAllowed(user *telebot.User) bool
	}
)

func New(service Service) *Plugin {
	return &Plugin{
		allowService: service,
	}
}

func (*Plugin) Name() string {
	return "allow"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/allow(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnAllow,
			AdminOnly:   true,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/deny(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnDeny,
			AdminOnly:   true,
			GroupOnly:   true,
		},
	}
}

func (plg *Plugin) OnAllow(c plugin.GobotContext) error {
	if c.Message().IsReply() { // Allow user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := plg.allowService.IsUserAllowed(c.Message().ReplyTo.Sender)
		if isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot bereits überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := plg.allowService.AllowUser(c.Message().ReplyTo.Sender)
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to allow user")
			return c.Reply("❌ Fehler beim Erlauben des Nutzers.", utils.DefaultSendOptions)
		}

		return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt überall benutzen",
			html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
			utils.DefaultSendOptions)
	} else { // Allow group
		isAllowed := plg.allowService.IsChatAllowed(c.Chat())

		if isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot bereits nutzen.", utils.DefaultSendOptions)
		}

		err := plg.allowService.AllowChat(c.Chat())
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to allow chat")
			return c.Reply("❌ Fehler beim Erlauben des Chats.", utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nutzen", utils.DefaultSendOptions)
	}
}

func (plg *Plugin) OnDeny(c plugin.GobotContext) error {
	if c.Message().IsReply() { // Deny user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := plg.allowService.IsUserAllowed(c.Message().ReplyTo.Sender)
		if !isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot nicht überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := plg.allowService.DenyUser(c.Message().ReplyTo.Sender)
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to deny user")
			return c.Reply("❌ Fehler beim Verweigern des Nutzers.", utils.DefaultSendOptions)
		}

		return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt nicht mehr überall benutzen",
			html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
			utils.DefaultSendOptions)
	} else { // Deny group
		isAllowed := plg.allowService.IsChatAllowed(c.Chat())

		if !isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot nicht nutzen.", utils.DefaultSendOptions)
		}

		err := plg.allowService.DenyChat(c.Chat())
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to deny chat")
			return c.Reply("❌ Fehler beim Verweigern des Chats.", utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nicht mehr nutzen", utils.DefaultSendOptions)
	}
}
