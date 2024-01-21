package bot

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/xid"
)

type Processor struct {
	allowService      model.AllowService
	chatsUsersService model.ChatsUsersService
	managerService    model.ManagerService
	userService       model.UserService
}

func NewProcessor(allowService model.AllowService, chatsUsersService model.ChatsUsersService, managerService model.ManagerService, userService model.UserService) *Processor {
	return &Processor{
		allowService:      allowService,
		chatsUsersService: chatsUsersService,
		managerService:    managerService,
		userService:       userService,
	}
}

func (p *Processor) ProcessUpdate(d *ext.Dispatcher, b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message != nil {

		if ctx.Message.LeftChatMember != nil {
			return p.onUserLeft(ctx)
		}

		if ctx.Message.NewChatMembers != nil {
			return p.onUserJoined(ctx)
		}

		return p.onMessage(b, ctx)
	}

	if ctx.EditedMessage != nil {
		return p.onMessage(b, ctx)
	}

	if ctx.CallbackQuery != nil {
		return p.onCallback(b, ctx)
	}

	if ctx.InlineQuery != nil {
		return p.onInlineQuery(b, ctx)
	}

	return nil
}

func (p *Processor) onMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	isEdited := msg.EditDate != 0

	isAllowed := p.allowService.IsUserAllowed(ctx.EffectiveUser)
	if utils.FromGroup(msg) && !isAllowed {
		isAllowed = p.allowService.IsChatAllowed(ctx.EffectiveChat)
	}

	if !isAllowed {
		log.Debug().Int64("chat_id", ctx.EffectiveChat.Id).Msg("User/Chat is not allowed")
		return nil
	}

	var err error

	if !isEdited {
		if utils.IsPrivate(msg) {
			err = p.userService.Create(ctx.EffectiveUser)
		} else {
			err = p.chatsUsersService.Create(ctx.EffectiveChat, ctx.EffectiveUser)
		}
		if err != nil {
			return err
		}
	}

	text := msg.Caption
	if text == "" {
		text = msg.Text
	}

	for _, plg := range p.managerService.Plugins() {
		plg := plg
		for _, h := range plg.Handlers(&b.User) {
			h := h

			handler, ok := h.(*plugin.CommandHandler)
			if !ok {
				continue
			}

			if isEdited && !handler.HandleEdits {
				continue
			}

			if !utils.FromGroup(msg) && handler.GroupOnly {
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
			// TODO: Other handler types
			default:
				panic("Unspported handler type!!")
			}

			if matched {
				log.Printf("Matched plugin '%s': %s (%T)", plg.Name(), handler.Trigger, handler.Trigger)

				if !p.managerService.IsPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					continue
				}

				if utils.FromGroup(msg) && p.managerService.IsPluginDisabledForChat(ctx.EffectiveChat, plg.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plg.Name())
					continue
				}

				if handler.AdminOnly && !utils.IsAdmin(ctx.EffectiveUser) {
					log.Print("User is not an admin.")
					continue
				}

				go func() {
					defer func() {
						if r := recover(); r != nil {
							guid := xid.New().String()
							log.Err(errors.New("panic")).
								Str("guid", guid).
								Int64("chat_id", ctx.EffectiveChat.Id).
								Int64("user_id", ctx.EffectiveUser.Id).
								Str("text", ctx.EffectiveMessage.Text).
								Str("component", plg.Name()).
								Msgf("%s", r)
							_, _ = ctx.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
						}
					}()
					err := handler.Run(b, plugin.GobotContext{
						Context:      ctx,
						Matches:      matches,
						NamedMatches: namedMatches,
					})
					if err != nil {
						guid := xid.New().String()
						log.Err(err).
							Str("guid", guid).
							Int64("chat_id", ctx.EffectiveChat.Id).
							Int64("user_id", ctx.EffectiveUser.Id).
							Str("text", ctx.EffectiveMessage.Text).
							Str("component", plg.Name()).
							Send()
						_, _ = ctx.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
					}
				}()

			}

		}
	}

	return nil
}

func (p *Processor) onCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	callback := ctx.CallbackQuery
	msg := callback.Message

	if callback.Data == "" {
		_, err := callback.Answer(b, nil)
		return err
	}

	isAllowed := p.allowService.IsUserAllowed(&ctx.CallbackQuery.From)
	if msg != nil && utils.FromGroup(msg) && !isAllowed {
		isAllowed = p.allowService.IsChatAllowed(ctx.EffectiveChat)
	}

	if !isAllowed {
		_, err := callback.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "Du darfst diesen Bot nicht nutzen.",
			ShowAlert: true,
		})
		return err
	}

	for _, plg := range p.managerService.Plugins() {
		plg := plg
		for _, h := range plg.Handlers(&b.User) {
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

				if !p.managerService.IsPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
						Text:      "Dieser Befehl ist nicht verfügbar.",
						ShowAlert: true,
					})
					return err
				}

				if utils.FromGroup(msg) && p.managerService.IsPluginDisabledForChat(ctx.EffectiveChat, plg.Name()) {
					log.Printf("Plugin %s is disabled for this chat", plg.Name())
					_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
						Text:      "Dieser Befehl ist nicht verfügbar.",
						ShowAlert: true,
					})
					return err
				}

				if handler.AdminOnly && !utils.IsAdmin(ctx.EffectiveUser) {
					log.Print("User is not an admin.")
					_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
						Text:      "Du bist kein Bot-Administrator.",
						ShowAlert: true,
					})
					return err
				}

				if handler.Cooldown > 0 {
					callbackTime := time.Unix(ctx.CallbackQuery.Message.GetDate(), 0)
					currentTime := time.Now()
					waitTime := handler.Cooldown - currentTime.Sub(callbackTime)

					if waitTime > 0 {
						waitTimeStr := fmt.Sprintf("%.1f", waitTime.Seconds())
						waittimeStr := strings.ReplaceAll(waitTimeStr, ".", ",")
						_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
							Text:      fmt.Sprintf("🕒 Bitte warte noch %s Sekunden.", waittimeStr),
							ShowAlert: true,
						})
						return err
					}
				}

				if handler.DeleteButton && ctx.EffectiveMessage != nil {
					go func() {
						_, _, err := ctx.EffectiveMessage.EditReplyMarkup(b, nil)
						if err != nil {
							log.Err(err).
								Int64("chat_id", ctx.EffectiveChat.Id).
								Msg("Error removing inline keyboard")
						}
					}()
				}

				namedMatches := make(map[string]string)
				for i, name := range matches {
					namedMatches[command.SubexpNames()[i]] = name
				}

				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Err(errors.New("panic")).
								Int64("chat_id", ctx.EffectiveChat.Id).
								Str("callback_data", callback.Data).
								Str("component", plg.Name()).
								Msgf("%s", r)
						}
					}()
					err := handler.Run(b, plugin.GobotContext{
						Context:      ctx,
						Matches:      matches,
						NamedMatches: namedMatches,
					})
					if err != nil {
						log.Err(err).
							Int64("chat_id", ctx.EffectiveChat.Id).
							Str("callback_data", callback.Data).
							Str("component", plg.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (p *Processor) onInlineQuery(b *gotgbot.Bot, ctx *ext.Context) error {
	inlineQuery := ctx.InlineQuery

	if inlineQuery.Query == "" {
		_, err := ctx.InlineQuery.Answer(b,
			nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  utils.InlineQueryFailureCacheTime,
				IsPersonal: true,
			})
		return err
	}

	for _, plg := range p.managerService.Plugins() {
		plg := plg
		for _, h := range plg.Handlers(&b.User) {
			h := h
			handler, ok := h.(*plugin.InlineHandler)
			if !ok {
				continue
			}

			command, ok := handler.Command().(*regexp.Regexp)
			if !ok {
				panic("Unsupported callback handler type!! Must be regexp.Regexp!")
			}

			matches := command.FindStringSubmatch(inlineQuery.Query)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)
				if !p.managerService.IsPluginEnabled(plg.Name()) {
					log.Printf("Plugin %s is disabled globally", plg.Name())
					_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
						CacheTime:  utils.InlineQueryFailureCacheTime,
						IsPersonal: true,
					})
					return err
				}

				if handler.AdminOnly && !utils.IsAdmin(ctx.EffectiveUser) {
					log.Print("User is not an admin.")
					_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
						CacheTime:  utils.InlineQueryFailureCacheTime,
						IsPersonal: true,
					})
					return err
				}

				if !handler.CanBeUsedByEveryone {
					isAllowed := p.allowService.IsUserAllowed(ctx.EffectiveUser)
					if !isAllowed {
						_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
							CacheTime:  utils.InlineQueryFailureCacheTime,
							IsPersonal: true,
						})
						return err
					}
				}

				namedMatches := make(map[string]string)
				for i, name := range matches {
					namedMatches[command.SubexpNames()[i]] = name
				}

				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Err(errors.New("panic")).
								Int64("user_id", ctx.EffectiveUser.Id).
								Str("query", ctx.InlineQuery.Query).
								Str("component", plg.Name()).
								Msgf("%s", r)
						}
					}()
					err := handler.Run(b, plugin.GobotContext{
						Context:      ctx,
						Matches:      matches,
						NamedMatches: namedMatches,
					})
					if err != nil {
						log.Err(err).
							Int64("user_id", ctx.EffectiveUser.Id).
							Str("query", ctx.InlineQuery.Query).
							Str("component", plg.Name()).
							Send()
					}
				}()

			}
		}
	}

	return nil
}

func (p *Processor) onUserJoined(ctx *ext.Context) error {
	return p.chatsUsersService.CreateBatch(ctx.EffectiveChat, &ctx.Message.NewChatMembers)
}

func (p *Processor) onUserLeft(ctx *ext.Context) error {
	if ctx.Message.LeftChatMember.IsBot {
		return nil
	}
	return p.chatsUsersService.Leave(ctx.EffectiveChat, ctx.Message.LeftChatMember)
}
