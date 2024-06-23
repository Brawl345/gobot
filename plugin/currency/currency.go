package currency

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("currency")

const (
	ApiUrl               = "https://api.frankfurter.app/latest?amount=%s&from=%s&to=%s"
	InlineQueryCacheTime = 3600
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "currency"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "cash",
			Description: "<Wert> <Basis> [Zu] - Währung umrechnen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/cash(?:@%s)? ([\d,]+) ([A-Za-z]{3}) (?:in )?([A-Za-z]{3})$`, botInfo.Username)),
			HandlerFunc: onConvertFromTo,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/cash(?:@%s)? ([\d,]+) ([A-Za-z]{3})$`, botInfo.Username)),
			HandlerFunc: onConvertToEUR,
		},
		&plugin.InlineHandler{
			Trigger:     regexp.MustCompile(`(?i)^cash ([\d,]+) ([A-Za-z]{3}) (?:in )?([A-Za-z]{3})$`),
			HandlerFunc: onConvertFromToInline,
		},
		&plugin.InlineHandler{
			Trigger:     regexp.MustCompile(`(?i)^cash ([\d,]+) ([A-Za-z]{3})$`),
			HandlerFunc: onConvertToEURInline,
		},
	}
}

func convertCurrency(amount, from, to string) (string, error) {
	amount = strings.ReplaceAll(amount, ",", ".")
	_, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", ErrBadAmount
	}

	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if from == to {
		return "", ErrSameCurrency
	}

	var response Response
	var httpError *httpUtils.HttpError
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      fmt.Sprintf(ApiUrl, amount, from, to),
		Response: &response,
	})
	if err != nil {
		if errors.As(err, &httpError) && httpError.StatusCode == http.StatusNotFound {
			return "", ErrBadCurrency
		}
		return "", err
	}

	amountStr := utils.FormatFloat(response.Amount)
	amountStr = strings.ReplaceAll(amountStr, ",00", "")
	toStr := utils.FormatFloat(response.Rates[to])
	toStr = strings.ReplaceAll(toStr, ",00", "")

	return fmt.Sprintf("💶 %s %s = <b>%s %s</b>", amountStr, from, toStr, to), nil
}

func onConvertFromTo(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)

	text, err := convertCurrency(c.Matches[1], c.Matches[2], c.Matches[3])
	if err != nil {
		switch {
		case errors.Is(err, ErrBadAmount):
			_, err := c.EffectiveMessage.Reply(b, "❌ Ungültiger Betrag", utils.DefaultSendOptions())
			return err
		case errors.Is(err, ErrBadCurrency):
			_, err := c.EffectiveMessage.Reply(b, "❌ Bitte gib eine <a href=\"https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.de.html\">gültige Währung</a> an.", utils.DefaultSendOptions())
			return err
		case errors.Is(err, ErrSameCurrency):
			_, err := c.EffectiveMessage.Reply(b, "❌ Die beiden Währungen sind identisch.", utils.DefaultSendOptions())
			return err
		default:
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("Failed to convert currency")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Daten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}
	}

	_, err = c.EffectiveMessage.Reply(b, text, utils.DefaultSendOptions())
	return err
}

func onConvertToEUR(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	text, err := convertCurrency(c.Matches[1], c.Matches[2], "EUR")
	if err != nil {
		switch {
		case errors.Is(err, ErrBadAmount):
			_, err := c.EffectiveMessage.Reply(b, "❌ Ungültiger Betrag", utils.DefaultSendOptions())
			return err
		case errors.Is(err, ErrBadCurrency):
			_, err := c.EffectiveMessage.Reply(b, "❌ Bitte gib eine <a href=\"https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.de.html\">gültige Zielwährung</a> an.", utils.DefaultSendOptions())
			return err
		case errors.Is(err, ErrSameCurrency):
			_, err := c.EffectiveMessage.Reply(b, "❌ Mit diesem Befehl rechnest du bereits in Euro um.", utils.DefaultSendOptions())
			return err
		default:
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("Failed to convert currency")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Abrufen der Daten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}
	}

	_, err = c.EffectiveMessage.Reply(b, text, utils.DefaultSendOptions())
	return err
}

func onConvertFromToInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	text, err := convertCurrency(c.Matches[1], c.Matches[2], c.Matches[3])

	if err != nil {
		log.Err(err).
			Msg("Failed to convert currency (inline mode)")
		_, err := c.InlineQuery.Answer(
			b,
			nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  utils.InlineQueryFailureCacheTime,
				IsPersonal: true,
			},
		)
		return err
	}

	title := strings.NewReplacer("💶 ", "", "<b>", "", "</b>", "").Replace(text)

	_, err = c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:    strconv.Itoa(rand.Int()),
				Title: title,
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText:        text,
					ParseMode:          gotgbot.ParseModeHTML,
					LinkPreviewOptions: &gotgbot.LinkPreviewOptions{IsDisabled: true},
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: InlineQueryCacheTime},
	)
	return err
}

func onConvertToEURInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	text, err := convertCurrency(c.Matches[1], c.Matches[2], "EUR")

	if err != nil {
		log.Err(err).
			Msg("Failed to convert currency (inline mode)")
		_, err := c.InlineQuery.Answer(
			b,
			nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  utils.InlineQueryFailureCacheTime,
				IsPersonal: true,
			},
		)
		return err
	}

	title := strings.NewReplacer("💶 ", "", "<b>", "", "</b>", "").Replace(text)
	_, err = c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:    strconv.Itoa(rand.Int()),
				Title: title,
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText:        text,
					ParseMode:          gotgbot.ParseModeHTML,
					LinkPreviewOptions: &gotgbot.LinkPreviewOptions{IsDisabled: true},
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: InlineQueryCacheTime},
	)
	return err
}
