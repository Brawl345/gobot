package bot

import (
	"errors"
)

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

func NewBasePlugin(bot *Nextbot) (*Plugin, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}
	return &Plugin{
		Bot: bot,
	}, nil
}
