package bot

import (
	"errors"
	"regexp"
)

type IPlugin interface {
	GetName() string
	GetCommandHandlers() []CommandHandler
	GetCallbackHandlers() []CallbackHandler
	GetInlineHandlers() []InlineHandler
	Init()
}

type CommandHandler struct {
	Command     any
	Handler     NextbotHandlerFunc
	AdminOnly   bool
	GroupOnly   bool
	HandleEdits bool
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
