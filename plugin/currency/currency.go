package currency

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "cash",
			Description: "<Wert> <Basis> [Zu] - Währung umrechnen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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
	err = httpUtils.GetRequest(fmt.Sprintf(ApiUrl, amount, from, to), &response)
	if err != nil {
		if errors.As(err, &httpError) && httpError.StatusCode == 404 {
			return "", ErrBadCurrency
		}
		return "", err
	}

	amountStr := fmt.Sprintf("%.2f", response.Amount)
	amountStr = strings.ReplaceAll(amountStr, ".", ",")
	amountStr = strings.ReplaceAll(amountStr, ",00", "")
	toStr := fmt.Sprintf("%.2f", response.Rates[to])
	toStr = strings.ReplaceAll(toStr, ".", ",")
	toStr = strings.ReplaceAll(toStr, ",00", "")

	return fmt.Sprintf("💶 %s %s = <b>%s %s</b>", amountStr, from, toStr, to), nil
}

func onConvertFromTo(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)

	text, err := convertCurrency(c.Matches[1], c.Matches[2], c.Matches[3])
	if err != nil {
		switch err {
		case ErrBadAmount:
			return c.Reply("❌ Ungültiger Betrag", utils.DefaultSendOptions)
		case ErrBadCurrency:
			return c.Reply("❌ Bitte gib eine <a href=\"https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.de.html\">gültige Währung</a> an.", utils.DefaultSendOptions)
		case ErrSameCurrency:
			return c.Reply("❌ Die beiden Währungen sind identisch.", utils.DefaultSendOptions)
		default:
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("Failed to convert currency")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Abrufen der Daten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		}
	}

	return c.Reply(text, utils.DefaultSendOptions)
}

func onConvertToEUR(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	text, err := convertCurrency(c.Matches[1], c.Matches[2], "EUR")
	if err != nil {
		switch err {
		case ErrBadAmount:
			return c.Reply("❌ Ungültiger Betrag", utils.DefaultSendOptions)
		case ErrBadCurrency:
			return c.Reply("❌ Bitte gib eine <a href=\"https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.de.html\">gültige Zielwährung</a> an.", utils.DefaultSendOptions)
		case ErrSameCurrency:
			return c.Reply("❌ Mit diesem Befehl rechnest du bereits in Euro um.", utils.DefaultSendOptions)
		default:
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Msg("Failed to convert currency")
			return c.Reply(fmt.Sprintf("❌ Fehler beim Abrufen der Daten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		}
	}

	return c.Reply(text, utils.DefaultSendOptions)
}

func onConvertFromToInline(c plugin.GobotContext) error {
	text, err := convertCurrency(c.Matches[1], c.Matches[2], c.Matches[3])

	if err != nil {
		log.Err(err).
			Msg("Failed to convert currency (inline mode)")
		return c.Answer(&telebot.QueryResponse{
			Results:    telebot.Results{},
			CacheTime:  utils.InlineQueryFailureCacheTime,
			IsPersonal: true,
		})
	}

	title := strings.NewReplacer("💶 ", "", "<b>", "", "</b>", "").Replace(text)
	result := &telebot.ArticleResult{
		Title: title,
		Text:  text,
	}
	result.SetContent(&telebot.InputTextMessageContent{
		Text:           text,
		ParseMode:      telebot.ModeHTML,
		DisablePreview: true,
	})

	return c.Answer(&telebot.QueryResponse{
		Results:    telebot.Results{result},
		CacheTime:  InlineQueryCacheTime,
		IsPersonal: false,
	})
}

func onConvertToEURInline(c plugin.GobotContext) error {
	text, err := convertCurrency(c.Matches[1], c.Matches[2], "EUR")

	if err != nil {
		log.Err(err).
			Msg("Failed to convert currency (inline mode)")
		return c.Answer(&telebot.QueryResponse{
			Results:    telebot.Results{},
			CacheTime:  utils.InlineQueryFailureCacheTime,
			IsPersonal: true,
		})
	}

	title := strings.NewReplacer("💶 ", "", "<b>", "", "</b>", "").Replace(text)
	result := &telebot.ArticleResult{
		Title: title,
		Text:  text,
	}
	result.SetContent(&telebot.InputTextMessageContent{
		Text:           text,
		ParseMode:      telebot.ModeHTML,
		DisablePreview: true,
	})

	return c.Answer(&telebot.QueryResponse{
		Results:    telebot.Results{result},
		CacheTime:  InlineQueryCacheTime,
		IsPersonal: false,
	})
}
