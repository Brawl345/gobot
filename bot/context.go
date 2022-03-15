package bot

import "gopkg.in/telebot.v3"

type NextbotContext struct {
	telebot.Context
	Matches []string
}

type NextbotHandlerFunc func(c NextbotContext) error
