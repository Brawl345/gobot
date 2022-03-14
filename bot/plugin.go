package bot

import (
	"gopkg.in/telebot.v3"
	"regexp"
)

// TODO: Kein Interface sondern struct...?
type IPlugin interface {
	GetName() string
	GetHandlers() []Handler
	Init()
}

type Handler struct {
	Command *regexp.Regexp // or interface{}?
	Handler telebot.HandlerFunc
}

type Plugin struct {
	name     string
	handlers []Handler
}

func NewPlugin(name string, handlers []Handler) *Plugin {
	if handlers == nil {
		panic("handlers can not be nil")
	}
	return &Plugin{
		name:     name,
		handlers: handlers,
	}
}

func (b *Plugin) GetName() string {
	return b.name
}

func (b *Plugin) GetHandlers() []Handler {
	return b.handlers
}

func (b *Plugin) Init() {
	panic("Init() Method not implemented!")
}
