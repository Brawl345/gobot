package quotes

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

type (
	Plugin struct {
		quoteService Service
	}

	Service interface {
		GetQuote(chat *gotgbot.Chat) (string, error)
		SaveQuote(chat *gotgbot.Chat, quote string) error
		DeleteQuote(chat *gotgbot.Chat, quote string) error
	}
)

var log = logger.New("quotes")

func New(quoteService Service) *Plugin {
	return &Plugin{
		quoteService: quoteService,
	}
}

func (p *Plugin) Name() string {
	return "quotes"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "quote",
			Description: "Zitat anzeigen",
		},
		{
			Command:     "addquote",
			Description: "<Zitat> - Zitat hinzufügen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/quote(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.getQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/addquote(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.addQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/save(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.addQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delquote(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.deleteQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delquote(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.deleteQuote,
			GroupOnly:   true,
		},
		&plugin.CallbackHandler{
			Trigger:      regexp.MustCompile(`^quotes_again$`),
			HandlerFunc:  p.getQuote,
			DeleteButton: true,
			Cooldown:     time.Second * 2,
		},
	}
}

func (p *Plugin) getQuote(b *gotgbot.Bot, c plugin.GobotContext) error {
	quote, err := p.quoteService.GetQuote(c.EffectiveChat)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "<b>Es wurden noch keine Zitate eingespeichert!</b>\n"+
				"Füge welche mit <code>/addquote ZITAT</code> hinzu.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("quote", quote).
			Msg("failed to save quote")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveChat.SendMessage(b, quote, &gotgbot.SendMessageOpts{
		LinkPreviewOptions:  &gotgbot.LinkPreviewOptions{IsDisabled: true},
		DisableNotification: true,
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "Nochmal",
						CallbackData: "quotes_again",
					},
				},
			},
		},
	})
	return err
}

func (p *Plugin) addQuote(b *gotgbot.Bot, c plugin.GobotContext) error {
	var quote string
	if tgUtils.IsReply(c.EffectiveMessage) &&
		!c.EffectiveSender.IsBot() {
		if c.EffectiveMessage.ReplyToMessage.Text != "" {
			quote = fmt.Sprintf("\"%s\" —%s", c.EffectiveMessage.ReplyToMessage.Text, c.Matches[1])
		} else if c.EffectiveMessage.ReplyToMessage.Caption != "" {
			quote = fmt.Sprintf("\"%s\" —%s", c.EffectiveMessage.ReplyToMessage.Caption, c.Matches[1])
		}
	}

	if quote == "" {
		quote = c.Matches[1]
	}

	err := p.quoteService.SaveQuote(c.EffectiveChat, quote)

	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			_, err := c.EffectiveMessage.Reply(b, "<b>💡 Zitat existiert bereits!</b>", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("quote", quote).
			Msg("failed to save quote")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "<b>✅ Gespeichert!</b>",
	})
}

func (p *Plugin) deleteQuote(b *gotgbot.Bot, c plugin.GobotContext) error {
	var quote string
	if len(c.Matches) > 1 {
		quote = c.Matches[1]
	} else {
		if !tgUtils.IsReply(c.EffectiveMessage) || c.EffectiveMessage.ReplyToMessage.Text == "" {
			return nil
		}
		quoteMatches := regexp.MustCompile(fmt.Sprintf(`(?i)^(?:/addquote(?:@%s)? )?([\s\S]+)$`, b.Username)).FindStringSubmatch(c.EffectiveMessage.ReplyToMessage.Text)
		if len(quoteMatches) < 2 {
			return nil
		}
		quote = quoteMatches[1]
	}

	err := p.quoteService.DeleteQuote(c.EffectiveChat, quote)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "<b>❌ Zitat nicht gefunden!</b>", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("quote", quote).
			Msg("failed to delete quote")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "<b>✅ Zitat gelöscht!</b>",
	})
}
