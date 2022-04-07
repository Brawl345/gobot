package bot

import "regexp"

type (
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
