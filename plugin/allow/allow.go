package allow

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("allow")

type (
	Plugin struct {
		allowService model.AllowService
	}
)

func New(service model.AllowService) *Plugin {
	return &Plugin{
		allowService: service,
	}
}

func (*Plugin) Name() string {
	return "allow"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil // Because it's a superuser command
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func (p *Plugin) OnAllow(b *gotgbot.Bot, c plugin.GobotContext) error {
	if utils.IsReply(c.EffectiveMessage) { // Allow user
		if c.EffectiveMessage.ReplyToMessage.From.IsBot {
			_, err := c.EffectiveMessage.Reply(b, "🤖🤖🤖", utils.DefaultSendOptions())
			return err
		}

		isAllowed := p.allowService.IsUserAllowed(c.EffectiveMessage.ReplyToMessage.From)
		if isAllowed {
			_, err := c.EffectiveMessage.Reply(b,
				fmt.Sprintf(
					"✅ <b>%s</b> darf den Bot bereits überall benutzen.",
					utils.Escape(c.EffectiveMessage.ReplyToMessage.From.FirstName),
				),
				utils.DefaultSendOptions(),
			)
			return err
		}

		err := p.allowService.AllowUser(c.EffectiveMessage.ReplyToMessage.From)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.EffectiveMessage.ReplyToMessage.From.Id).
				Msg("Failed to allow user")
			_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Erlauben des Nutzers.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt überall benutzen",
			utils.Escape(c.EffectiveMessage.ReplyToMessage.From.FirstName)),
			utils.DefaultSendOptions())
		return err
	} else { // Allow group
		isAllowed := p.allowService.IsChatAllowed(c.EffectiveChat)

		if isAllowed {
			_, err := c.EffectiveMessage.Reply(b, "✅ Dieser Chat darf den Bot bereits nutzen.", utils.DefaultSendOptions())
			return err
		}

		err := p.allowService.AllowChat(c.EffectiveChat)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.EffectiveMessage.ReplyToMessage.From.Id).
				Msg("Failed to allow chat")

			_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Erlauben des Chats.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		_, err = c.EffectiveMessage.Reply(b, "✅ Dieser Chat darf den Bot jetzt nutzen", utils.DefaultSendOptions())
		return err
	}
}

func (p *Plugin) OnDeny(b *gotgbot.Bot, c plugin.GobotContext) error {
	if utils.IsReply(c.EffectiveMessage) { // Deny user
		if c.EffectiveMessage.ReplyToMessage.From.IsBot {
			_, err := c.EffectiveMessage.Reply(b, "🤖🤖🤖", utils.DefaultSendOptions())
			return err
		}

		isAllowed := p.allowService.IsUserAllowed(c.EffectiveMessage.ReplyToMessage.From)
		if !isAllowed {
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("✅ <b>%s</b> darf den Bot nicht überall benutzen.",
				utils.Escape(c.EffectiveMessage.ReplyToMessage.From.FirstName)),
				utils.DefaultSendOptions())
			return err
		}

		err := p.allowService.DenyUser(c.EffectiveMessage.ReplyToMessage.From)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.EffectiveMessage.ReplyToMessage.From.Id).
				Msg("Failed to deny user")

			_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Verweigern des Nutzers.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("✅ <b>%s</b> darf den Bot jetzt nicht mehr überall benutzen",
			utils.Escape(c.EffectiveMessage.ReplyToMessage.From.FirstName)),
			utils.DefaultSendOptions())
		return err
	} else { // Deny group
		isAllowed := p.allowService.IsChatAllowed(c.EffectiveChat)

		if !isAllowed {
			_, err := c.EffectiveMessage.Reply(b, "✅ Dieser Chat darf den Bot nicht nutzen.", utils.DefaultSendOptions())
			return err
		}

		err := p.allowService.DenyChat(c.EffectiveChat)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Int64("chat_id", c.EffectiveMessage.ReplyToMessage.From.Id).
				Msg("Failed to deny chat")
			_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Verweigern des Chats.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		_, err = c.EffectiveMessage.Reply(b, "✅ Dieser Chat darf den Bot jetzt nicht mehr nutzen", utils.DefaultSendOptions())
		return err
	}
}
