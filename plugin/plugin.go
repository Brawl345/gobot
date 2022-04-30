package plugin

import (
	"regexp"
	"time"

	"gopkg.in/telebot.v3"
)

type (
	Plugin interface {
		Name() string
		Handlers(botInfo *telebot.User) []Handler
	}

	Handler interface {
		Command() any
		Run(c GobotContext) error
	}

	GobotContext struct {
		telebot.Context
		Matches      []string          // Regex matches
		NamedMatches map[string]string // Named Regex matches
	}

	GobotHandlerFunc func(c GobotContext) error

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

func (h *CommandHandler) Run(c GobotContext) error {
	return h.HandlerFunc(c)
}

func (h *CallbackHandler) Command() any {
	return h.Trigger
}

func (h *CallbackHandler) Run(c GobotContext) error {
	return h.HandlerFunc(c)
}

func (h *InlineHandler) Command() any {
	return h.Trigger
}

func (h *InlineHandler) Run(c GobotContext) error {
	return h.HandlerFunc(c)
}
