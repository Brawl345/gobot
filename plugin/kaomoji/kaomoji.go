package kaomoji

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
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

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "lf",
			Description: lennyFace,
		},
		{
			Command:     "lod",
			Description: lookOfDisapproval,
		},
		{
			Command:     "shrug",
			Description: shrug,
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)/(?:shrug|nbc|idc)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onShrug,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)/lf(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onLennyFace,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)/lod(?:@%s)?$`, botInfo.Username)),
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

func onShrug(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := b.SendMessage(c.EffectiveChat.Id, shrug, utils.DefaultSendOptions())
	return err
}

func onShrugInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:    strconv.Itoa(rand.Int()),
				Title: shrug,
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText: shrug,
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: InlineQueryCacheTime},
	)
	return err
}

func onLennyFace(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := b.SendMessage(c.EffectiveChat.Id, lennyFace, utils.DefaultSendOptions())
	return err
}

func onLennyFaceInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:    strconv.Itoa(rand.Int()),
				Title: lennyFace,
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText: lennyFace,
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: InlineQueryCacheTime},
	)
	return err
}

func onLookOfDisapproval(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := b.SendMessage(c.EffectiveChat.Id, lookOfDisapproval, utils.DefaultSendOptions())
	return err
}

func onLookOfDisapprovalInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:    strconv.Itoa(rand.Int()),
				Title: lookOfDisapproval,
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText: lookOfDisapproval,
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: InlineQueryCacheTime},
	)
	return err
}
