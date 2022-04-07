package bot

import (
	"regexp"

	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

func (bot *Nextbot) OnText(c telebot.Context) error {
	msg := c.Message()
	isEdited := msg.LastEdit != 0

	isAllowed := bot.IsUserAllowed(c.Sender())
	if msg.FromGroup() && !isAllowed {
		isAllowed = bot.IsChatAllowed(c.Chat())
	}

	if !isAllowed {
		return nil
	}

	var err error

	if !isEdited {
		if msg.Private() {
			err = bot.DB.Users.Create(c.Sender())
		} else {
			err = bot.DB.ChatsUsers.Create(c.Chat(), c.Sender())
		}
		if err != nil {
			return err
		}
	}

	text := msg.Caption
	if text == "" {
		text = msg.Text
	}

	for _, plugin := range bot.plugins {
		plugin := plugin
		for _, h := range plugin.Handlers(bot.Me) {
			h := h

			handler, ok := h.(*CommandHandler)
			if !ok {
				continue
			}

			if isEdited && !handler.HandleEdits {
				continue
			}

			if !msg.FromGroup() && handler.GroupOnly {
				continue
			}

			var matched bool
			var matches []string

			switch command := handler.Command().(type) {
			case *regexp.Regexp:
				matches = command.FindStringSubmatch(text)
				matched = len(matches) > 0
			case string:
				switch {
				// More to be added when needed
				case msg.Animation != nil:
					matched = command == telebot.OnAnimation
				case msg.Photo != nil:
					matched = command == telebot.OnPhoto
				case msg.Document != nil:
					matched = command == telebot.OnDocument
				case msg.Sticker != nil:
					matched = command == telebot.OnSticker
				}

				if !matched && msg.Media() != nil {
					matched = command == telebot.OnMedia
				}
			default:
				panic("Unspported BaseHandler type!!")
			}

			if matched {
				log.Printf("Matched plugin %s: %s", plugin.Name(), handler.Trigger)

				if !bot.isPluginEnabled(plugin.Name()) {
					log.Printf("Plugin %s is disabled globally", plugin.Name())
					continue
				}

				if msg.FromGroup() && bot.isPluginDisabledForChat(c.Chat(), plugin.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plugin.Name())
					continue
				}

				if handler.AdminOnly && !utils.IsAdmin(c.Sender()) {
					log.Print("User is not an admin.")
					continue
				}

				go func() {
					err := handler.Run(NextbotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plugin.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (bot *Nextbot) OnCallback(c telebot.Context) error {
	msg := c.Message()
	callback := c.Callback()

	if callback.Data == "" {
		return c.Respond()
	}

	isAllowed := bot.IsUserAllowed(c.Sender())
	if msg.FromGroup() && !isAllowed {
		isAllowed = bot.IsChatAllowed(c.Chat())
	}

	if !isAllowed {
		return c.Respond(&telebot.CallbackResponse{
			Text:      "Du darfst diesen Bot nicht nutzen.",
			ShowAlert: true,
		})
	}

	for _, plugin := range bot.plugins {
		plugin := plugin
		for _, h := range plugin.Handlers(bot.Me) {
			h := h

			handler, ok := h.(*CallbackHandler)
			if !ok {
				continue
			}

			command, ok := handler.Command().(*regexp.Regexp)
			if !ok {
				panic("Unsupported callback BaseHandler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(callback.Data)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plugin.Name(), handler.Trigger)

				if !bot.isPluginEnabled(plugin.Name()) {
					log.Printf("Plugin %s is disabled globally", plugin.Name())
					return c.Respond(&telebot.CallbackResponse{
						Text:      "Dieser Befehl ist nicht verfügbar.",
						ShowAlert: true,
					})
				}

				if msg.FromGroup() && bot.isPluginDisabledForChat(c.Chat(), plugin.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plugin.Name())
					return c.Respond(&telebot.CallbackResponse{
						Text:      "Dieser Befehl ist nicht verfügbar.",
						ShowAlert: true,
					})
				}

				if handler.AdminOnly && !utils.IsAdmin(c.Sender()) {
					log.Print("User is not an admin.")
					return c.Respond(&telebot.CallbackResponse{
						Text:      "Du bist kein Bot-Administrator.",
						ShowAlert: true,
					})
				}

				go func() {
					err := handler.Run(NextbotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plugin.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (bot *Nextbot) OnInlineQuery(c telebot.Context) error {
	inlineQuery := c.Query()

	if inlineQuery.Text == "" {
		return c.Answer(&telebot.QueryResponse{
			CacheTime:  1,
			IsPersonal: true,
		})
	}

	for _, plugin := range bot.plugins {
		plugin := plugin
		for _, h := range plugin.Handlers(bot.Me) {
			h := h
			handler, ok := h.(*InlineHandler)
			if !ok {
				continue
			}

			command, ok := handler.Command().(*regexp.Regexp)
			if !ok {
				panic("Unsupported callback BaseHandler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(inlineQuery.Text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plugin.Name(), handler.Trigger)
				if !bot.isPluginEnabled(plugin.Name()) {
					log.Printf("Plugin %s is disabled globally", plugin.Name())
					return c.Answer(&telebot.QueryResponse{
						CacheTime:  1,
						IsPersonal: true,
					})
				}

				if handler.AdminOnly && !utils.IsAdmin(c.Sender()) {
					log.Print("User is not an admin.")
					return c.Answer(&telebot.QueryResponse{
						CacheTime:  1,
						IsPersonal: true,
					})
				}

				if !handler.CanBeUsedByEveryone {
					isAllowed := bot.IsUserAllowed(c.Sender())
					if !isAllowed {
						return c.Answer(&telebot.QueryResponse{
							CacheTime:  1,
							IsPersonal: true,
						})
					}
				}

				go func() {
					err := handler.Run(NextbotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plugin.Name()).
							Send()
					}
				}()

			}
		}
	}

	return c.Answer(&telebot.QueryResponse{
		CacheTime:  1,
		IsPersonal: true,
	})

}

func (bot *Nextbot) OnUserJoined(c telebot.Context) error {
	return bot.DB.ChatsUsers.CreateBatch(c.Chat(), &c.Message().UsersJoined)
}

func (bot *Nextbot) OnUserLeft(c telebot.Context) error {
	if c.Message().UserLeft.IsBot {
		return nil
	}
	return bot.DB.ChatsUsers.Leave(c.Chat(), c.Message().UserLeft)
}

// NullRoute is a special route that just ignores the message
// but will still fire middleware
func (bot *Nextbot) NullRoute(_ telebot.Context) error {
	return nil
}
