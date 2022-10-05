package calc

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

const (
	ApiUrl               = "http://api.mathjs.org/v4/?expr=%s"
	InlineQueryCacheTime = 7200
)

var log = logger.New("calc")

type (
	Plugin struct{}

	ApiError struct {
		Message string
	}
)

func (e *ApiError) Error() string {
	return strings.ReplaceAll(e.Message, "Error: ", "")
}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "calc"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "calc",
			Description: "<Ausdruck> - Taschenrechner",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/calc(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onCalc,
		},
		&plugin.InlineHandler{
			Trigger:     regexp.MustCompile(`(?i)^calc (.+)$`),
			HandlerFunc: onCalcInline,
		},
	}
}

func calculate(expr string) (string, error) {
	expr = strings.ReplaceAll(expr, ",", ".")

	var err error

	resp, err := httpUtils.HttpClient.Get(fmt.Sprintf(ApiUrl, url.QueryEscape(expr)))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return "", &httpUtils.HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("failed to close response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := string(body)

	if resp.StatusCode == 400 {
		return "", &ApiError{Message: result}
	}

	result = strings.ReplaceAll(string(body), ".", ",")
	return result, nil
}

func onCalc(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)

	result, err := calculate(c.Matches[1])
	if err != nil {
		var apiError *ApiError
		if errors.As(err, &apiError) {
			return c.Reply(fmt.Sprintf("❌ <b>Fehler:</b> <i>%s</i>", utils.Escape(apiError.Error())),
				utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("failed to calculate")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply(fmt.Sprintf("= %s", result), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
	})
}

func onCalcInline(c plugin.GobotContext) error {
	result, err := calculate(c.Matches[1])

	if err != nil {
		var apiError *ApiError
		if errors.As(err, &apiError) {
			log.Debug().Err(err).Msg("user input fail")
		} else {
			log.Err(err).Msg("failed to calculate")
		}
		return c.Answer(&telebot.QueryResponse{
			Results:    telebot.Results{},
			CacheTime:  utils.InlineQueryFailureCacheTime,
			IsPersonal: true,
		})
	}

	return c.Answer(&telebot.QueryResponse{
		Results: telebot.Results{&telebot.ArticleResult{
			Title:       c.Matches[1],
			Description: fmt.Sprintf("= %s", result),
			Text:        fmt.Sprintf("%s = %s", c.Matches[1], result),
		}},
		CacheTime: InlineQueryCacheTime,
	})
}
