package plugin

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"regexp"
	"time"
)

type (
	Plugin interface {
		Name() string

		// Commands will be shown in the menu button
		Commands() []gotgbot.BotCommand

		// Handlers are used to react to specific strings & entities in a message
		Handlers(botInfo *gotgbot.User) []Handler
	}

	Handler interface {
		Command() any
		Run(b *gotgbot.Bot, c GobotContext) error
	}

	GobotContext struct {
		*ext.Context
		Matches      []string          // Regex matches
		NamedMatches map[string]string // Named Regex matches
	}

	GobotHandlerFunc func(b *gotgbot.Bot, c GobotContext) error

	CommandHandler struct {
		Trigger     any
		HandlerFunc GobotHandlerFunc
		AdminOnly   bool
		GroupOnly   bool
		HandleEdits bool
	}

	CallbackHandler struct {
		HandlerFunc  GobotHandlerFunc
		Trigger      *regexp.Regexp
		AdminOnly    bool
		DeleteButton bool
		Cooldown     time.Duration
	}

	InlineHandler struct {
		HandlerFunc         GobotHandlerFunc
		Trigger             *regexp.Regexp
		AdminOnly           bool
		CanBeUsedByEveryone bool
	}
)

func (h *CommandHandler) Command() any {
	return h.Trigger
}

func (h *CommandHandler) Run(b *gotgbot.Bot, c GobotContext) error {
	return h.HandlerFunc(b, c)
}

func (h *CallbackHandler) Command() any {
	return h.Trigger
}

func (h *CallbackHandler) Run(b *gotgbot.Bot, c GobotContext) error {
	return h.HandlerFunc(b, c)
}

func (h *InlineHandler) Command() any {
	return h.Trigger
}

func (h *InlineHandler) Run(b *gotgbot.Bot, c GobotContext) error {
	return h.HandlerFunc(b, c)
}
