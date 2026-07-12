package bot

import (
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type (
	regexpCommandEntry struct {
		plugin  plugin.Plugin
		handler *plugin.CommandHandler
		regexp  *regexp.Regexp
	}

	mediaCommandEntry struct {
		plugin  plugin.Plugin
		handler *plugin.CommandHandler
		trigger tgUtils.MessageTrigger
	}

	entityCommandEntry struct {
		plugin  plugin.Plugin
		handler *plugin.CommandHandler
		entity  tgUtils.EntityType
	}

	callbackEntry struct {
		plugin  plugin.Plugin
		handler *plugin.CallbackHandler
	}

	inlineEntry struct {
		plugin  plugin.Plugin
		handler *plugin.InlineHandler
	}

	handlerRegistry struct {
		regexpCommands []regexpCommandEntry
		mediaCommands  []mediaCommandEntry
		entityCommands []entityCommandEntry
		callbacks      []callbackEntry
		inlines        []inlineEntry
	}
)

// buildHandlerRegistry compiles every plugin's handlers once and buckets them by
// type and trigger kind so updates dispatch without recompiling regexes.
func buildHandlerRegistry(plugins []plugin.Plugin, botInfo *gotgbot.User) *handlerRegistry {
	r := &handlerRegistry{}

	for _, plg := range plugins {
		for _, h := range plg.Handlers(botInfo) {
			switch handler := h.(type) {
			case *plugin.CommandHandler:
				switch command := handler.Command().(type) {
				case *regexp.Regexp:
					r.regexpCommands = append(r.regexpCommands, regexpCommandEntry{plg, handler, command})
				case tgUtils.MessageTrigger:
					r.mediaCommands = append(r.mediaCommands, mediaCommandEntry{plg, handler, command})
				case tgUtils.EntityType:
					r.entityCommands = append(r.entityCommands, entityCommandEntry{plg, handler, command})
				default:
					panic("Unsupported handler type!!")
				}
			case *plugin.CallbackHandler:
				r.callbacks = append(r.callbacks, callbackEntry{plg, handler})
			case *plugin.InlineHandler:
				r.inlines = append(r.inlines, inlineEntry{plg, handler})
			}
		}
	}

	return r
}
