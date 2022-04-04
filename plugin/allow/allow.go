package allow

import (
	"fmt"
	"html"
	"regexp"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
)

var log = logger.NewLogger("allow")

type Plugin struct {
	*bot.Plugin
}

func (*Plugin) GetName() string {
	return "allow"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/allow(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler:   plg.OnAllow,
			AdminOnly: true,
			GroupOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/deny(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler:   plg.OnDeny,
			AdminOnly: true,
			GroupOnly: true,
		},
	}
}

func (plg *Plugin) OnAllow(c bot.NextbotContext) error {
	if c.Message().IsReply() { // Allow user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := plg.Bot.IsUserAllowed(c.Message().ReplyTo.Sender)
		if isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot bereits überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := plg.Bot.AllowUser(c.Message().ReplyTo.Sender)
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
		isAllowed := plg.Bot.IsChatAllowed(c.Chat())

		if isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot bereits nutzen.", utils.DefaultSendOptions)
		}

		err := plg.Bot.AllowChat(c.Chat())
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to allow chat")
			return c.Reply("❌ Fehler beim Erlauben des Chats.", utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nutzen", utils.DefaultSendOptions)
	}
}

func (plg *Plugin) OnDeny(c bot.NextbotContext) error {
	if c.Message().IsReply() { // Deny user
		if c.Message().ReplyTo.Sender.IsBot {
			return c.Reply("🤖🤖🤖", utils.DefaultSendOptions)
		}

		isAllowed := plg.Bot.IsUserAllowed(c.Message().ReplyTo.Sender)
		if !isAllowed {
			return c.Reply(fmt.Sprintf("✅ <b>%s</b> darf den Bot nicht überall benutzen.",
				html.EscapeString(c.Message().ReplyTo.Sender.FirstName)),
				utils.DefaultSendOptions)
		}

		err := plg.Bot.DenyUser(c.Message().ReplyTo.Sender)
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
		isAllowed := plg.Bot.IsChatAllowed(c.Chat())

		if !isAllowed {
			return c.Reply("✅ Dieser Chat darf den Bot nicht nutzen.", utils.DefaultSendOptions)
		}

		err := plg.Bot.DenyChat(c.Chat())
		if err != nil {
			log.Err(err).
				Int64("chat_id", c.Message().ReplyTo.Sender.ID).
				Msg("Failed to deny chat")
			return c.Reply("❌ Fehler beim Verweigern des Chats.", utils.DefaultSendOptions)
		}

		return c.Reply("✅ Dieser Chat darf den Bot jetzt nicht mehr nutzen", utils.DefaultSendOptions)
	}
}
