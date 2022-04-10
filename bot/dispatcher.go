package bot

import (
	"regexp"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/plugin/allow"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

type Dispatcher struct {
	allowService      allow.Service
	chatsUsersService models.ChatsUsersService
	managerService    *managerService
	userService       models.UserService
}

func (d *Dispatcher) OnText(c telebot.Context) error {
	msg := c.Message()
	isEdited := msg.LastEdit != 0

	isAllowed := d.allowService.IsUserAllowed(c.Sender())
	if msg.FromGroup() && !isAllowed {
		isAllowed = d.allowService.IsChatAllowed(c.Chat())
	}

	if !isAllowed {
		return nil
	}

	var err error

	if !isEdited {
		if msg.Private() {
			err = d.userService.Create(c.Sender())
		} else {
			err = d.chatsUsersService.Create(c.Chat(), c.Sender())
		}
		if err != nil {
			return err
		}
	}

	text := msg.Caption
	if text == "" {
		text = msg.Text
	}

	for _, plg := range d.managerService.plugins {
		plg := plg
		for _, h := range plg.Handlers(c.Bot().Me) {
			h := h

			handler, ok := h.(*plugin.CommandHandler)
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
				log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)

				if !d.managerService.isPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					continue
				}

				if msg.FromGroup() && d.managerService.isPluginDisabledForChat(c.Chat(), plg.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plg.Name())
					continue
				}

				if handler.AdminOnly && !utils.IsAdmin(c.Sender()) {
					log.Print("User is not an admin.")
					continue
				}

				go func() {
					err := handler.Run(plugin.GobotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plg.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (d *Dispatcher) OnCallback(c telebot.Context) error {
	msg := c.Message()
	callback := c.Callback()

	if callback.Data == "" {
		return c.Respond()
	}

	isAllowed := d.allowService.IsUserAllowed(c.Sender())
	if msg.FromGroup() && !isAllowed {
		isAllowed = d.allowService.IsChatAllowed(c.Chat())
	}

	if !isAllowed {
		return c.Respond(&telebot.CallbackResponse{
			Text:      "Du darfst diesen Bot nicht nutzen.",
			ShowAlert: true,
		})
	}

	for _, plg := range d.managerService.plugins {
		plg := plg
		for _, h := range plg.Handlers(c.Bot().Me) {
			h := h

			handler, ok := h.(*plugin.CallbackHandler)
			if !ok {
				continue
			}

			command, ok := handler.Command().(*regexp.Regexp)
			if !ok {
				panic("Unsupported callback BaseHandler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(callback.Data)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)

				if !d.managerService.isPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					return c.Respond(&telebot.CallbackResponse{
						Text:      "Dieser Befehl ist nicht verfügbar.",
						ShowAlert: true,
					})
				}

				if msg.FromGroup() && d.managerService.isPluginDisabledForChat(c.Chat(), plg.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plg.Name())
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
					err := handler.Run(plugin.GobotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plg.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (d *Dispatcher) OnInlineQuery(c telebot.Context) error {
	inlineQuery := c.Query()

	if inlineQuery.Text == "" {
		return c.Answer(&telebot.QueryResponse{
			CacheTime:  1,
			IsPersonal: true,
		})
	}

	for _, plg := range d.managerService.plugins {
		plg := plg
		for _, h := range plg.Handlers(c.Bot().Me) {
			h := h
			handler, ok := h.(*plugin.InlineHandler)
			if !ok {
				continue
			}

			command, ok := handler.Command().(*regexp.Regexp)
			if !ok {
				panic("Unsupported callback BaseHandler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(inlineQuery.Text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)
				if !d.managerService.isPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
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
					isAllowed := d.allowService.IsUserAllowed(c.Sender())
					if !isAllowed {
						return c.Answer(&telebot.QueryResponse{
							CacheTime:  1,
							IsPersonal: true,
						})
					}
				}

				go func() {
					err := handler.Run(plugin.GobotContext{
						Context: c,
						Matches: matches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", c.Sender().ID).
							Str("text", c.Text()).
							Str("component", plg.Name()).
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

func (d *Dispatcher) OnUserJoined(c telebot.Context) error {
	return d.chatsUsersService.CreateBatch(c.Chat(), &c.Message().UsersJoined)
}

func (d *Dispatcher) OnUserLeft(c telebot.Context) error {
	if c.Message().UserLeft.IsBot {
		return nil
	}
	return d.chatsUsersService.Leave(c.Chat(), c.Message().UserLeft)
}

// NullRoute is a special route that just ignores the message
// but will still fire middleware
func (d *Dispatcher) NullRoute(_ telebot.Context) error {
	return nil
}
