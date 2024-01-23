package expand

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
)

var log = logger.New("expand")

const (
	MaxDepth           = 7
	MaxLinksPerMessage = 3
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "expand"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "expand",
			Description: "<URL> - Link entkürzen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/expand(?:@%s)? .+$`, botInfo.Username)),
			HandlerFunc: onExpand,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/expand(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onExpandFromReply,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)(?:bit\.ly|bitly\.com|j\.mp|tinyurl.com)/.+`),
			HandlerFunc: onExpand,
		},
	}
}

func expandUrl(url string) (string, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", utils.UserAgent)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.Header.Get("Location") == "" {
		return "", &httpUtils.HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	return resp.Header.Get("Location"), nil
}

func loop(sb *strings.Builder, url string, depth int) {
	expandedUrl, err := expandUrl(url)
	if err != nil {
		var httpErr *httpUtils.HttpError
		if errors.As(err, &httpErr) {
			sb.WriteString(fmt.Sprintf("➡ <b>%s</b>\n", httpErr.Status))
			return
		}
		sb.WriteString("❌ <b>Nicht erreichbar</b>\n")
		log.Err(err).
			Str("url", url).
			Msg("Error expanding url")
		return
	}
	sb.WriteString(fmt.Sprintf("➡ %s\n", expandedUrl))
	if depth >= MaxDepth {
		sb.WriteString("➡ ...\n")
		return
	} else {
		loop(sb, expandedUrl, depth+1)
	}
}

func onExpand(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

	var shortUrls []string
	for _, entity := range utils.AnyEntities(c.Message()) {
		if entity.Type == telebot.EntityURL {
			shortUrls = append(shortUrls, c.Message().EntityText(entity))
		} else if entity.Type == telebot.EntityTextLink {
			shortUrls = append(shortUrls, entity.URL)
		}
	}

	if len(shortUrls) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "Keine Links gefunden", utils.DefaultSendOptions)
		return err
	}

	var limitExceeded bool
	if len(shortUrls) > MaxLinksPerMessage {
		shortUrls = shortUrls[:MaxLinksPerMessage]
		limitExceeded = true
	}

	var sb strings.Builder

	for _, url := range shortUrls {
		sb.WriteString(fmt.Sprintf("%s\n", url))
		loop(&sb, url, 1)
		sb.WriteString("\n")
	}

	if limitExceeded {
		sb.WriteString("💡 <i>...weitere Links ignoriert</i>\n")
	}

	_, err := c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}

func onExpandFromReply(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !c.Message().IsReply() {
		log.Debug().
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Msg("Message is not a reply")
		return nil
	}

	if strings.HasPrefix(c.Message().ReplyTo.Text, "/expand") ||
		strings.HasPrefix(c.Message().ReplyTo.Caption, "/expand") {
		_, err := c.EffectiveMessage.Reply(b, "😠", utils.DefaultSendOptions)
		return err
	}

	var shortUrls []string
	for _, entity := range utils.AnyEntities(c.Message().ReplyTo) {
		if entity.Type == telebot.EntityURL {
			shortUrls = append(shortUrls, c.Message().ReplyTo.EntityText(entity))
		} else if entity.Type == telebot.EntityTextLink {
			shortUrls = append(shortUrls, entity.URL)
		}
	}

	if len(shortUrls) == 0 {
		_, err := c.Bot().Reply(c.Message().ReplyTo, "Keine Links gefunden", utils.DefaultSendOptions)
		return err
	}

	var limitExceeded bool
	if len(shortUrls) > MaxLinksPerMessage {
		shortUrls = shortUrls[:MaxLinksPerMessage]
		limitExceeded = true
	}

	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)
	var sb strings.Builder

	for _, url := range shortUrls {
		sb.WriteString(fmt.Sprintf("%s\n", url))
		loop(&sb, url, 1)
		sb.WriteString("\n")
	}

	if limitExceeded {
		sb.WriteString("💡 <i>...weitere Links ignoriert</i>\n")
	}

	_, err := c.Bot().Reply(c.Message().ReplyTo, sb.String(), utils.DefaultSendOptions)
	return err
}
