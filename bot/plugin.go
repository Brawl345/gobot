package bot

import (
	"gopkg.in/telebot.v3"
	"regexp"
)

type NextbotHandlerFunc func(b *Nextbot, c telebot.Context) error

type Handler struct {
	Command *regexp.Regexp
	Handler NextbotHandlerFunc
}

type Plugin struct {
	Name     string
	Handlers []Handler
}
