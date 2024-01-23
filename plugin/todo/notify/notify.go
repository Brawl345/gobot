package notify

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

var log = logger.New("notify")

type (
	Plugin struct {
		notifyService Service
	}

	Service interface {
		Enabled(chat *telebot.Chat, user *telebot.User) (bool, error)
		Enable(chat *telebot.Chat, user *telebot.User) error
		GetAllToBeNotifiedUsers(chat *telebot.Chat, mentionedUsernames []string) ([]int64, error)
		Disable(chat *telebot.Chat, user *telebot.User) error
	}
)

func New(notifyService Service) *Plugin {
	return &Plugin{
		notifyService: notifyService,
	}
}

func (p *Plugin) Name() string {
	return "notify"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "notify",
			Description: "Über neue Erwähnungen informiert werden",
		},
		{
			Command:     "notify_disable",
			Description: "Nicht mehr über neue Erwähnungen informiert werden",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/notify(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.enableNotify,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/notify_disable(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.disableNotify,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     telebot.EntityMention,
			HandlerFunc: p.notify,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) notify(b *gotgbot.Bot, c plugin.GobotContext) error {
	var mentionedUsernames []string
	for _, entity := range utils.AnyEntities(c.Message()) {
		if entity.Type == telebot.EntityMention {
			username := strings.TrimPrefix(c.Message().EntityText(entity), "@")
			username = strings.ToLower(username)
			if !slices.Contains(mentionedUsernames, username) && username !=
				strings.ToLower(c.Sender().Username) {
				mentionedUsernames = append(mentionedUsernames, username)
			}
		}
	}

	if len(mentionedUsernames) == 0 {
		return nil
	}

	userIDs, err := p.notifyService.GetAllToBeNotifiedUsers(c.Chat(), mentionedUsernames)
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Msg("error while getting all usernames that should be notified")
		return nil
	}

	if len(userIDs) == 0 {
		return nil
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"🔔 <b>%s</b> hat dich erwähnt:\n",
			utils.Escape(utils.FullName(c.Sender().FirstName, c.Sender().LastName)),
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"👥 <b>%s</b> | 📅 %s | 🕒 %s Uhr\n",
			utils.Escape(c.Chat().Title),
			c.Message().Time().Format("02.01.2006"),
			c.Message().Time().Format("15:04:05"),
		),
	)
	sb.WriteString(utils.Escape(c.Message().Text))
	if c.Message().Text == "" {
		sb.WriteString(utils.Escape(c.Message().Caption))
	}

	for _, userID := range userIDs {
		_, err := c.Bot().Send(telebot.ChatID(userID), sb.String(), utils.DefaultSendOptions)

		if err != nil {
			if errors.Is(err, telebot.ErrBlockedByUser) {
				log.Warn().
					Int64("to_user_id", userID).
					Msg("User blocked the bot")
			} else if errors.Is(err, telebot.ErrNotStartedByUser) {
				log.Warn().
					Int64("to_user_id", userID).
					Msg("User didn't start the bot")
			} else if errors.Is(err, telebot.ErrUserIsDeactivated) {
				log.Warn().
					Int64("to_user_id", userID).
					Msg("User account is deactivated")
			} else {
				log.Err(err).
					Int64("to_user_id", userID).
					Msg("error while sending notification")
			}
		}
	}

	return nil
}

func (p *Plugin) enableNotify(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.Sender().Username == "" {
		_, err := c.EffectiveMessage.Reply(b, "😕 Du benötigst einen Benutzernamen um dieses Feature zu nutzen.", utils.DefaultSendOptions)
		return err
	}

	testMsg, err := c.Bot().Send(c.Sender(), "✅", utils.DefaultSendOptions)
	if err != nil {
		if errors.Is(err, telebot.ErrBlockedByUser) {
			_, err := c.EffectiveMessage.Reply(b, "😭 Du hast mich blockiert T__T", utils.DefaultSendOptions)
			return err
		} else if errors.Is(err, telebot.ErrNotStartedByUser) {
			_, err := c.EffectiveMessage.Reply(b, "ℹ Bitte starte mich vor dem Aktivieren zuerst privat.", utils.DefaultSendOptions)
			return err
		}
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error while sending test message")
		return c.Reply(fmt.Sprintf("❌ Ich wollte dir eine Nachricht senden, aber das hat nicht funktioniert Bitte den Administrator des Bots um Hilfe und sende ihm folgenden Fehler-Code:%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	err = c.Bot().Delete(testMsg)
	if err != nil {
		log.Err(err).
			Msg("error while deleting test message, lmao")
	}

	enabled, err := p.notifyService.Enabled(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error during enabled check")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if enabled {
		return c.Reply("💡 Du wirst in dieser Gruppe schon über neue Erwähnungen informiert.")
	}

	err = p.notifyService.Enable(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error while enabling notifications")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	return c.Reply("✅ Du wirst jetzt über neue Erwähnungen in dieser Gruppe informiert!\n"+
		"Nutze <code>/notify_disable</code> zum Deaktivieren.", utils.DefaultSendOptions)
}

func (p *Plugin) disableNotify(b *gotgbot.Bot, c plugin.GobotContext) error {
	enabled, err := p.notifyService.Enabled(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error during enabled check")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if !enabled {
		return c.Reply("💡 Du wirst in dieser Gruppe nicht über neue Erwähnungen informiert.",
			utils.DefaultSendOptions)
	}

	err = p.notifyService.Disable(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error while disabling notifications")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	return c.Reply("✅ Du wirst nicht mehr über neue Erwähnungen in dieser Gruppe informiert.",
		utils.DefaultSendOptions)
}
