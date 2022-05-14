package quotes

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

type (
	Plugin struct {
		quoteService Service
	}

	Service interface {
		GetQuote(chat *telebot.Chat) (string, error)
		SaveQuote(chat *telebot.Chat, quote string) error
		DeleteQuote(chat *telebot.Chat, quote string) error
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "quote",
			Description: "Zitat anzeigen",
		},
		{
			Text:        "addquote",
			Description: "<Zitat> - Zitat hinzufügen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/quote(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.getQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/addquote(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.addQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delquote(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.deleteQuote,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delquote(?:@%s)? (.+)$`, botInfo.Username)),
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

func (p *Plugin) getQuote(c plugin.GobotContext) error {
	quote, err := p.quoteService.GetQuote(c.Chat())
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Reply("<b>Es wurden noch keine Zitate eingespeichert!</b>\n"+
				"Füge welche mit <code>/addquote ZITAT</code> hinzu.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Str("quote", quote).
			Msg("failed to save quote")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Send(quote, &telebot.SendOptions{
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ReplyMarkup: &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{
					{
						Text: "Nochmal",
						Data: "quotes_again",
					},
				},
			},
		},
	})
}

func (p *Plugin) addQuote(c plugin.GobotContext) error {
	// TODO: Save by reply
	quote := c.Matches[1]
	err := p.quoteService.SaveQuote(c.Chat(), quote)

	if err != nil {
		if errors.Is(err, models.ErrAlreadyExists) {
			return c.Reply("<b>💡 Zitat existiert bereits!</b>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Str("quote", quote).
			Msg("failed to save quote")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("<b>✅ Gespeichert!</b>", utils.DefaultSendOptions)
}

func (p *Plugin) deleteQuote(c plugin.GobotContext) error {
	var quote string
	if len(c.Matches) > 1 {
		quote = c.Matches[1]
	} else {
		if !c.Message().IsReply() || c.Message().ReplyTo.Text == "" {
			return nil
		}
		quoteMatches := regexp.MustCompile(fmt.Sprintf(`(?i)^(?:/addquote(?:@%s)? )?(.+)$`, c.Bot().Me.Username)).FindStringSubmatch(c.Message().ReplyTo.Text)
		if len(quoteMatches) < 2 {
			return nil
		}
		quote = quoteMatches[1]
	}

	err := p.quoteService.DeleteQuote(c.Chat(), quote)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Reply("<b>❌ Zitat nicht gefunden!</b>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Str("quote", quote).
			Msg("failed to delete quote")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("<b>✅ Zitat gelöscht!</b>", utils.DefaultSendOptions)
}
