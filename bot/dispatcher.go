package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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
		log.Debug().Int64("chat_id", c.Chat().ID).Msg("User/Chat is not allowed")
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
			namedMatches := make(map[string]string)

			switch command := handler.Command().(type) {
			case *regexp.Regexp:
				matches = command.FindStringSubmatch(text)
				matched = len(matches) > 0
				if matched {
					for i, name := range matches {
						namedMatches[command.SubexpNames()[i]] = name
					}
				}
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
				case msg.Location != nil:
					matched = command == telebot.OnLocation
				case msg.Venue != nil:
					matched = command == telebot.OnVenue
				}

				if !matched && msg.Media() != nil {
					matched = command == telebot.OnMedia
				}
			case telebot.EntityType:
				entities := msg.Entities
				if entities == nil {
					entities = msg.CaptionEntities
				}
				for _, entity := range entities {
					matched = entity.Type == command
					if matched {
						break
					}
				}
			default:
				panic("Unspported handler type!!")
			}

			if matched {
				log.Printf("Matched plugin '%s': %s (%T)", plg.Name(), handler.Trigger, handler.Trigger)

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
						Context:      c,
						Matches:      matches,
						NamedMatches: namedMatches,
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
				panic("Unsupported callback handler type!! Must be regexp.Regexp!")
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

				if handler.Cooldown > 0 {
					callbackTime := c.Callback().Message.Time()
					currentTime := time.Now()
					waitTime := handler.Cooldown - currentTime.Sub(callbackTime)

					if waitTime > 0 {
						waitTimeStr := fmt.Sprintf("%.1f", waitTime.Seconds())
						waittimeStr := strings.ReplaceAll(waitTimeStr, ".", ",")
						return c.Respond(&telebot.CallbackResponse{
							Text:      fmt.Sprintf("🕒 Bitte warte noch %s Sekunden.", waittimeStr),
							ShowAlert: true,
						})
					}
				}

				if handler.DeleteButton && c.Message() != nil {
					go func() {
						err := c.Edit(&telebot.ReplyMarkup{})
						if err != nil {
							log.Err(err).
								Int64("chat_id", c.Sender().ID).
								Msg("Error removing inline keyboard")
						}
					}()
				}

				namedMatches := make(map[string]string)
				for i, name := range matches {
					namedMatches[command.SubexpNames()[i]] = name
				}

				go func() {
					err := handler.Run(plugin.GobotContext{
						Context:      c,
						Matches:      matches,
						NamedMatches: namedMatches,
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
			CacheTime:  utils.InlineQueryFailureCacheTime,
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
				panic("Unsupported callback handler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(inlineQuery.Text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)
				if !d.managerService.isPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					return c.Answer(&telebot.QueryResponse{
						CacheTime:  utils.InlineQueryFailureCacheTime,
						IsPersonal: true,
					})
				}

				if handler.AdminOnly && !utils.IsAdmin(c.Sender()) {
					log.Print("User is not an admin.")
					return c.Answer(&telebot.QueryResponse{
						CacheTime:  utils.InlineQueryFailureCacheTime,
						IsPersonal: true,
					})
				}

				if !handler.CanBeUsedByEveryone {
					isAllowed := d.allowService.IsUserAllowed(c.Sender())
					if !isAllowed {
						return c.Answer(&telebot.QueryResponse{
							CacheTime:  utils.InlineQueryFailureCacheTime,
							IsPersonal: true,
						})
					}
				}

				namedMatches := make(map[string]string)
				for i, name := range matches {
					namedMatches[command.SubexpNames()[i]] = name
				}

				go func() {
					err := handler.Run(plugin.GobotContext{
						Context:      c,
						Matches:      matches,
						NamedMatches: namedMatches,
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
