package calc

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
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

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "calc",
			Description: "<Ausdruck> - Taschenrechner",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

	var resp string
	var errorResp string
	var httpError *httpUtils.HttpError

	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:        httpUtils.MethodGet,
		URL:           fmt.Sprintf(ApiUrl, url.QueryEscape(expr)),
		Response:      &resp,
		ErrorResponse: &errorResp,
	})

	if err != nil {
		if errors.As(err, &httpError) && httpError.StatusCode == http.StatusBadRequest {
			return "", &ApiError{Message: errorResp}
		}

		return "", err
	}

	result := strings.ReplaceAll(resp, ".", ",")
	return result, nil
}

func onCalc(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	result, err := calculate(c.Matches[1])
	if err != nil {
		var apiError *ApiError
		if errors.As(err, &apiError) {
			_, err = c.EffectiveMessage.Reply(b,
				fmt.Sprintf("❌ <b>Fehler:</b> <i>%s</i>", utils.Escape(apiError.Error())),
				utils.DefaultSendOptions(),
			)
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("failed to calculate")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("= %s", result), &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
		DisableNotification: true,
	})
	return err
}

func onCalcInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	result, err := calculate(c.Matches[1])

	if err != nil {
		var apiError *ApiError
		if errors.As(err, &apiError) {
			log.Debug().Err(err).Msg("user input fail")
		} else {
			log.Err(err).Msg("failed to calculate")
		}
		_, err := c.InlineQuery.Answer(
			b,
			nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  utils.Ptr(utils.InlineQueryFailureCacheTime),
				IsPersonal: true,
			},
		)
		return err
	}

	_, err = c.InlineQuery.Answer(
		b,
		[]gotgbot.InlineQueryResult{
			gotgbot.InlineQueryResultArticle{
				Id:          strconv.Itoa(rand.Int()),
				Title:       c.Matches[1],
				Description: fmt.Sprintf("= %s", result),
				InputMessageContent: gotgbot.InputTextMessageContent{
					MessageText: fmt.Sprintf("%s = %s", c.Matches[1], result),
				},
			},
		},
		&gotgbot.AnswerInlineQueryOpts{CacheTime: utils.Ptr(int64(InlineQueryCacheTime))},
	)
	return err
}
