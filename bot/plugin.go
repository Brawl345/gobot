package bot

import (
	"errors"
	"regexp"
)

type IPlugin interface {
	GetName() string
	GetHandlers() []Handler
	Init()
}

type Handler struct {
	Command   *regexp.Regexp
	Handler   NextbotHandlerFunc
	AdminOnly bool
	GroupOnly bool
}

type Plugin struct {
	Bot *Nextbot
}

func (*Plugin) Init() {}

func NewPlugin(bot *Nextbot) (*Plugin, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}
	return &Plugin{
		Bot: bot,
	}, nil
}
