package bot

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/xid"
)

type Processor struct {
	allowService      model.AllowService
	chatsUsersService model.ChatsUsersService
	managerService    model.ManagerService
	userService       model.UserService
	shouldPrintMsgs   bool
	registryOnce      sync.Once
	registry          *handlerRegistry
}

// handlers returns the cached handler registry, building it on first use once
// the bot's identity and plugin list are known.
func (p *Processor) handlers(b *gotgbot.Bot) *handlerRegistry {
	p.registryOnce.Do(func() {
		p.registry = buildHandlerRegistry(p.managerService.Plugins(), &b.User)
	})
	return p.registry
}

func NewProcessor(allowService model.AllowService, chatsUsersService model.ChatsUsersService, managerService model.ManagerService, userService model.UserService) *Processor {
	_, shouldPrintMsgs := os.LookupEnv("PRINT_MSGS")
	return &Processor{
		allowService:      allowService,
		chatsUsersService: chatsUsersService,
		managerService:    managerService,
		userService:       userService,
		shouldPrintMsgs:   shouldPrintMsgs,
	}
}

func (p *Processor) ProcessUpdate(d *ext.Dispatcher, b *gotgbot.Bot, ctx *ext.Context) error {

	if p.shouldPrintMsgs {
		PrintMessage(ctx)
	}

	if ctx.GetType() == gotgbot.UpdateTypeMessage {

		if ctx.Message.LeftChatMember != nil {
			return p.onUserLeft(ctx)
		}

		if ctx.Message.NewChatMembers != nil {
			return p.onUserJoined(ctx)
		}

		if ctx.Message.NewChatTitle != "" || ctx.Message.NewChatPhoto != nil {
			return nil
		}

		return p.onMessage(b, ctx)
	}

	if ctx.GetType() == gotgbot.UpdateTypeEditedMessage {
		return p.onMessage(b, ctx)
	}

	if ctx.GetType() == gotgbot.UpdateTypeCallbackQuery {
		return p.onCallback(b, ctx)
	}

	if ctx.GetType() == gotgbot.UpdateTypeInlineQuery {
		return p.onInlineQuery(b, ctx)
	}

	return nil
}

func (p *Processor) onMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	isEdited := msg.EditDate != 0

	isAllowed := p.allowService.IsUserAllowed(ctx.EffectiveUser)
	if tgUtils.FromGroup(msg) && !isAllowed {
		isAllowed = p.allowService.IsChatAllowed(ctx.EffectiveChat)
	}

	if !isAllowed {
		log.Debug().Int64("chat_id", ctx.EffectiveChat.Id).Msg("User/Chat is not allowed")
		return nil
	}

	var err error

	if !isEdited {
		if tgUtils.IsPrivate(msg) {
			err = p.userService.Create(ctx.EffectiveUser)
		} else {
			err = p.chatsUsersService.Create(ctx.EffectiveChat, ctx.EffectiveUser)
		}
		if err != nil {
			return err
		}
	}

	text := msg.GetText()
	registry := p.handlers(b)

	for _, e := range registry.regexpCommands {
		if !commandApplies(e.handler, msg, isEdited) {
			continue
		}
		matches := e.regexp.FindStringSubmatch(text)
		if len(matches) == 0 {
			continue
		}
		p.dispatchCommand(b, ctx, e.plugin, e.handler, matches, namedMatchesOf(e.regexp, matches))
	}

	for _, e := range registry.mediaCommands {
		if !commandApplies(e.handler, msg, isEdited) {
			continue
		}
		if !mediaMatches(e.trigger, msg) {
			continue
		}
		p.dispatchCommand(b, ctx, e.plugin, e.handler, nil, map[string]string{})
	}

	for _, e := range registry.entityCommands {
		if !commandApplies(e.handler, msg, isEdited) {
			continue
		}
		if !entityMatches(e.entity, msg) {
			continue
		}
		p.dispatchCommand(b, ctx, e.plugin, e.handler, nil, map[string]string{})
	}

	return nil
}

func commandApplies(handler *plugin.CommandHandler, msg *gotgbot.Message, isEdited bool) bool {
	if isEdited && !handler.HandleEdits {
		return false
	}
	if !tgUtils.FromGroup(msg) && handler.GroupOnly {
		return false
	}
	return true
}

func mediaMatches(command tgUtils.MessageTrigger, msg *gotgbot.Message) bool {
	var matched bool
	switch {
	// More to be added when needed
	case msg.Document != nil:
		matched = command == tgUtils.DocumentMsg
	case len(msg.Photo) > 0:
		matched = command == tgUtils.PhotoMsg
	case msg.Location != nil:
		matched = command == tgUtils.LocationMsg
	case msg.Venue != nil:
		matched = command == tgUtils.VenueMsg
	case msg.Voice != nil:
		matched = command == tgUtils.VoiceMsg
	}

	if !matched && tgUtils.ContainsMedia(msg) {
		matched = command == tgUtils.AnyMedia
	}

	if !matched {
		matched = command == tgUtils.AnyMsg
	}

	return matched
}

func entityMatches(command tgUtils.EntityType, msg *gotgbot.Message) bool {
	entities := msg.Entities
	if entities == nil {
		entities = msg.CaptionEntities
	}

	for _, entity := range entities {
		if tgUtils.EntityType(entity.Type) == command {
			return true
		}
	}
	return false
}

func namedMatchesOf(command *regexp.Regexp, matches []string) map[string]string {
	namedMatches := make(map[string]string)
	for i, name := range matches {
		namedMatches[command.SubexpNames()[i]] = name
	}
	return namedMatches
}

func (p *Processor) dispatchCommand(b *gotgbot.Bot, ctx *ext.Context, plg plugin.Plugin, handler *plugin.CommandHandler, matches []string, namedMatches map[string]string) {
	log.Printf("Matched plugin '%s': %s (%T)", plg.Name(), handler.Trigger, handler.Trigger)

	if !p.managerService.IsPluginEnabled(plg.Name()) {
		log.Printf("Plugin %s is disabled globally", plg.Name())
		return
	}

	if tgUtils.FromGroup(ctx.EffectiveMessage) && p.managerService.IsPluginDisabledForChat(ctx.EffectiveChat, plg.Name()) {
		log.Printf("Plugin %s is disabled for this chat", plg.Name())
		return
	}

	if handler.AdminOnly && !tgUtils.IsAdmin(ctx.EffectiveUser) {
		log.Print("User is not an admin.")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				guid := xid.New().String()
				log.Err(errors.New("panic")).
					Str("guid", guid).
					Interface("ctx", ctx).
					Str("component", plg.Name()).
					Msgf("%s", r)
				_, _ = ctx.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
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
				Interface("ctx", ctx).
				Str("component", plg.Name()).
				Send()
			_, _ = ctx.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		}
	}()
}

func (p *Processor) onCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	callback := ctx.CallbackQuery
	msg := callback.Message

	if callback.Data == "" {
		_, err := callback.Answer(b, nil)
		return err
	}

	isAllowed := p.allowService.IsUserAllowed(&ctx.CallbackQuery.From)
	if msg != nil && tgUtils.FromGroup(msg) && !isAllowed {
		isAllowed = p.allowService.IsChatAllowed(ctx.EffectiveChat)
	}

	if !isAllowed {
		_, err := callback.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "Du darfst diesen Bot nicht nutzen.",
			ShowAlert: true,
		})
		return err
	}

	for _, e := range p.handlers(b).callbacks {
		plg := e.plugin
		handler := e.handler
		command := handler.Trigger

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

			if msg != nil && tgUtils.FromGroup(msg) && p.managerService.IsPluginDisabledForChat(ctx.EffectiveChat, plg.Name()) {
				log.Printf("Plugin %s is disabled for this chat", plg.Name())
				_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "Dieser Befehl ist nicht verfügbar.",
					ShowAlert: true,
				})
				return err
			}

			if handler.AdminOnly && !tgUtils.IsAdmin(ctx.EffectiveUser) {
				log.Print("User is not an admin.")
				_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "Du bist kein Bot-Administrator.",
					ShowAlert: true,
				})
				return err
			}

			if handler.Cooldown > 0 && msg != nil {
				callbackTime := utils.TimestampToTime(ctx.CallbackQuery.Message.GetDate())
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

			namedMatches := namedMatchesOf(command, matches)

			var chatId int64
			if ctx.EffectiveChat != nil {
				chatId = ctx.EffectiveChat.Id
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Err(errors.New("panic")).
							Int64("chat_id", chatId).
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
						Int64("chat_id", chatId).
						Str("callback_data", callback.Data).
						Str("component", plg.Name()).
						Send()
				}
			}()

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
				CacheTime:  utils.Ptr(utils.InlineQueryFailureCacheTime),
				IsPersonal: true,
			})
		return err
	}

	for _, e := range p.handlers(b).inlines {
		plg := e.plugin
		handler := e.handler
		command := handler.Trigger

		matches := command.FindStringSubmatch(inlineQuery.Query)
		if len(matches) > 0 {
			log.Printf("Matched plugin %s: %s", plg.Name(), handler.Trigger)
			if !p.managerService.IsPluginEnabled(plg.Name()) {
				log.Printf("Plugin %s is disabled globally", plg.Name())
				_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
					CacheTime:  utils.Ptr(utils.InlineQueryFailureCacheTime),
					IsPersonal: true,
				})
				return err
			}

			if handler.AdminOnly && !tgUtils.IsAdmin(ctx.EffectiveUser) {
				log.Print("User is not an admin.")
				_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
					CacheTime:  utils.Ptr(utils.InlineQueryFailureCacheTime),
					IsPersonal: true,
				})
				return err
			}

			if !handler.CanBeUsedByEveryone {
				isAllowed := p.allowService.IsUserAllowed(ctx.EffectiveUser)
				if !isAllowed {
					_, err := ctx.InlineQuery.Answer(b, nil, &gotgbot.AnswerInlineQueryOpts{
						CacheTime:  utils.Ptr(utils.InlineQueryFailureCacheTime),
						IsPersonal: true,
					})
					return err
				}
			}

			namedMatches := namedMatchesOf(command, matches)

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
