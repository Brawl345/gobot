package kaomoji

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

const (
	lennyFace         = "( ͡° ͜ʖ ͡°)"
	lookOfDisapproval = "ಠ_ಠ"
	shrug             = "¯\\_(ツ)_/¯"
)

const InlineQueryCacheTime = 7200

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "kaomoji"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "lf",
			Description: lennyFace,
		},
		{
			Text:        "lod",
			Description: lookOfDisapproval,
		},
		{
			Text:        "shrug",
			Description: shrug,
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/(?:shrug|nbc|idc)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onShrug,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/lf(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onLennyFace,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/lod(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onLookOfDisapproval,
		},
		&plugin.InlineHandler{
			Trigger:             regexp.MustCompile("(?i)^(?:shrug|nbc|idc)$"),
			CanBeUsedByEveryone: true,
			HandlerFunc:         onShrugInline,
		},
		&plugin.InlineHandler{
			Trigger:             regexp.MustCompile("(?i)^lf$"),
			CanBeUsedByEveryone: true,
			HandlerFunc:         onLennyFaceInline,
		},
		&plugin.InlineHandler{
			Trigger:             regexp.MustCompile("(?i)^lod$"),
			CanBeUsedByEveryone: true,
			HandlerFunc:         onLookOfDisapprovalInline,
		},
	}
}

func onShrug(c plugin.GobotContext) error {
	return c.Send(shrug, utils.DefaultSendOptions)
}

func onShrugInline(c plugin.GobotContext) error {
	return c.Answer(&telebot.QueryResponse{
		Results: telebot.Results{&telebot.ArticleResult{
			Title: shrug,
			Text:  shrug,
		}},
		CacheTime: InlineQueryCacheTime,
	})
}

func onLennyFace(c plugin.GobotContext) error {
	return c.Send(lennyFace, utils.DefaultSendOptions)
}

func onLennyFaceInline(c plugin.GobotContext) error {
	return c.Answer(&telebot.QueryResponse{
		Results: telebot.Results{&telebot.ArticleResult{
			Title: lennyFace,
			Text:  lennyFace,
		}},
		CacheTime: InlineQueryCacheTime,
	})
}

func onLookOfDisapproval(c plugin.GobotContext) error {
	return c.Send(lookOfDisapproval, utils.DefaultSendOptions)
}

func onLookOfDisapprovalInline(c plugin.GobotContext) error {
	return c.Answer(&telebot.QueryResponse{
		Results: telebot.Results{&telebot.ArticleResult{
			Title: lookOfDisapproval,
			Text:  lookOfDisapproval,
		}},
		CacheTime: InlineQueryCacheTime,
	})
}
