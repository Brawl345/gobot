package plugin

import (
	"regexp"

	"gopkg.in/telebot.v3"
)

type (
	Plugin interface {
		Name() string
		Handlers(botInfo *telebot.User) []Handler
	}

	Handler interface {
		Command() any
		Run(c NextbotContext) error
	}

	NextbotContext struct {
		telebot.Context
		Matches []string // Regex matches
	}

	NextbotHandlerFunc func(c NextbotContext) error

	CommandHandler struct {
		Trigger     any
		HandlerFunc NextbotHandlerFunc
		AdminOnly   bool
		GroupOnly   bool
		HandleEdits bool
	}

	CallbackHandler struct {
		HandlerFunc NextbotHandlerFunc
		Trigger     *regexp.Regexp
		AdminOnly   bool
	}

	InlineHandler struct {
		HandlerFunc         NextbotHandlerFunc
		Trigger             *regexp.Regexp
		AdminOnly           bool
		CanBeUsedByEveryone bool
	}
)

func (h *CommandHandler) Command() any {
	return h.Trigger
}

func (h *CommandHandler) Run(c NextbotContext) error {
	return h.HandlerFunc(c)
}

func (h *CallbackHandler) Command() any {
	return h.Trigger
}

func (h *CallbackHandler) Run(c NextbotContext) error {
	return h.HandlerFunc(c)
}

func (h *InlineHandler) Command() any {
	return h.Trigger
}

func (h *InlineHandler) Run(c NextbotContext) error {
	return h.HandlerFunc(c)
}
