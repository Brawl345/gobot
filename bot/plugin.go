package bot

import (
	"errors"
	"regexp"
)

type IPlugin interface {
	GetName() string
	GetHandlers() []Handler
	GetCallbackHandlers() []CallbackHandler
	GetInlineHandlers() []InlineHandler
	Init()
}

type Handler struct {
	Command   *regexp.Regexp
	Handler   NextbotHandlerFunc
	AdminOnly bool
	GroupOnly bool
}

type CallbackHandler struct {
	Command   *regexp.Regexp
	Handler   NextbotHandlerFunc
	AdminOnly bool
}

type InlineHandler struct {
	Command             *regexp.Regexp
	Handler             NextbotHandlerFunc
	AdminOnly           bool
	CanBeUsedByEveryone bool
}

type Plugin struct {
	Bot *Nextbot
}

func (*Plugin) Init() {}

func (*Plugin) GetCallbackHandlers() []CallbackHandler {
	return []CallbackHandler{}
}

func (*Plugin) GetInlineHandlers() []InlineHandler {
	return []InlineHandler{}
}

func NewPlugin(bot *Nextbot) (*Plugin, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}
	return &Plugin{
		Bot: bot,
	}, nil
}
