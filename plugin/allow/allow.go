package allow

import (
	"fmt"
	"html"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("allow")

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

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/allow(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnAllow,
			AdminOnly:   true,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/deny(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnDeny,
			AdminOnly:   true,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) OnAllow(c plugin.GobotContext) error {
	if c.Message().IsReply() { // Allow user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := p.allowService.IsUserAllowed(c.Message().ReplyTo.Sender)
		if isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot bereits überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := p.allowService.AllowUser(c.Message().ReplyTo.Sender)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to allow user")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Erlauben des Nutzers.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
		}

		return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt überall benutzen",
			html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
			utils.DefaultSendOptions)
	} else { // Allow group
		isAllowed := p.allowService.IsChatAllowed(c.Chat())

		if isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot bereits nutzen.", utils.DefaultSendOptions)
		}

		err := p.allowService.AllowChat(c.Chat())
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to allow chat")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Erlauben des Chats.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nutzen", utils.DefaultSendOptions)
	}
}

func (p *Plugin) OnDeny(c plugin.GobotContext) error {
	if c.Message().IsReply() { // Deny user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := p.allowService.IsUserAllowed(c.Message().ReplyTo.Sender)
		if !isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot nicht überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := p.allowService.DenyUser(c.Message().ReplyTo.Sender)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to deny user")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Verweigern des Nutzers.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
		}

		return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt nicht mehr überall benutzen",
			html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
			utils.DefaultSendOptions)
	} else { // Deny group
		isAllowed := p.allowService.IsChatAllowed(c.Chat())

		if !isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot nicht nutzen.", utils.DefaultSendOptions)
		}

		err := p.allowService.DenyChat(c.Chat())
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to deny chat")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Verweigern des Chats.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nicht mehr nutzen", utils.DefaultSendOptions)
	}
}
