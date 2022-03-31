package bot

import "gopkg.in/telebot.v3"

type NextbotContext struct {
	telebot.Context
	Matches []string // Regex matches
}

type NextbotHandlerFunc func(c NextbotContext) error
