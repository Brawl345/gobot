package notify

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	tgUtils "github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
	"golang.org/x/exp/slices"
)

var log = logger.New("notify")

type (
	Plugin struct {
		notifyService Service
	}

	Service interface {
		Enabled(chat *gotgbot.Chat, user *gotgbot.User) (bool, error)
		Enable(chat *gotgbot.Chat, user *gotgbot.User) error
		GetAllToBeNotifiedUsers(chat *gotgbot.Chat, mentionedUsernames []string) ([]int64, error)
		Disable(chat *gotgbot.Chat, user *gotgbot.User) error
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
			Trigger:     tgUtils.EntityTypeMention,
			HandlerFunc: p.notify,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) notify(b *gotgbot.Bot, c plugin.GobotContext) error {
	var mentionedUsernames []string

	for _, entity := range tgUtils.ParseAnyEntityTypes(c.EffectiveMessage, []tgUtils.EntityType{tgUtils.EntityTypeMention}) {
		username := strings.TrimPrefix(entity.Text, "@")
		username = strings.ToLower(username)
		if !slices.Contains(mentionedUsernames, username) && username !=
			strings.ToLower(c.EffectiveUser.Username) {
			mentionedUsernames = append(mentionedUsernames, username)
		}
	}

	if len(mentionedUsernames) == 0 {
		return nil
	}

	userIDs, err := p.notifyService.GetAllToBeNotifiedUsers(c.EffectiveChat, mentionedUsernames)
	if err != nil {
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
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
			utils.Escape(utils.FullName(c.EffectiveUser.FirstName, c.EffectiveUser.LastName)),
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"👥 <b>%s</b> | 📅 %s | 🕒 %s Uhr\n",
			utils.Escape(c.EffectiveChat.Title),
			utils.TimestampToTime(c.EffectiveMessage.Date).Format("02.01.2006"),
			utils.TimestampToTime(c.EffectiveMessage.Date).Format("15:04:05"),
		),
	)
	sb.WriteString(utils.Escape(c.EffectiveMessage.Text))
	if c.EffectiveMessage.Text == "" {
		sb.WriteString(utils.Escape(c.EffectiveMessage.Caption))
	}

	for _, userID := range userIDs {
		_, err := b.SendMessage(userID, sb.String(), utils.DefaultSendOptions())

		if err != nil {
			var telegramErr *gotgbot.TelegramError

			if errors.As(err, &telegramErr) {
				switch telegramErr.Description {
				case tgUtils.ErrBlockedByUser:
					log.Warn().
						Int64("to_user_id", userID).
						Msg("User blocked the bot")
				case tgUtils.ErrNotStartedByUser:
					log.Warn().
						Int64("to_user_id", userID).
						Msg("User didn't start the bot")
				case tgUtils.ErrUserIsDeactivated:
					log.Warn().
						Int64("to_user_id", userID).
						Msg("User account is deactivated")
				default:
					log.Err(err).
						Int64("to_user_id", userID).
						Msg("error while sending notification")
				}
			}
		}
	}

	return nil
}

func (p *Plugin) enableNotify(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.EffectiveUser.Username == "" {
		_, err := c.EffectiveMessage.ReplyMessage(b, "😕 Du benötigst einen Benutzernamen um dieses Feature zu nutzen.", utils.DefaultSendOptions())
		return err
	}

	testMsg, err := b.SendMessage(c.EffectiveUser.Id, "✅", utils.DefaultSendOptions())
	if err != nil {
		var telegramErr *gotgbot.TelegramError

		if errors.As(err, &telegramErr) {
			switch telegramErr.Description {
			case tgUtils.ErrBlockedByUser:
				_, err := c.EffectiveMessage.ReplyMessage(b, "😭 Du hast mich blockiert T__T", utils.DefaultSendOptions())
				return err
			case tgUtils.ErrNotStartedByUser:
				_, err := c.EffectiveMessage.ReplyMessage(b, "ℹ Bitte starte mich vor dem Aktivieren zuerst privat.", utils.DefaultSendOptions())
				return err
			}
		}

		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error while sending test message")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Ich wollte dir eine Nachricht senden, aber das hat nicht funktioniert Bitte den Administrator des Bots um Hilfe und sende ihm folgenden Fehler-Code:%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	_, err = testMsg.Delete(b, nil)
	if err != nil {
		log.Err(err).
			Msg("error while deleting test message, lmao")
	}

	enabled, err := p.notifyService.Enabled(c.EffectiveChat, c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error during enabled check")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if enabled {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Du wirst in dieser Gruppe schon über neue Erwähnungen informiert.", utils.DefaultSendOptions())
		return err
	}

	err = p.notifyService.Enable(c.EffectiveChat, c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error while enabling notifications")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Du wirst jetzt über neue Erwähnungen in dieser Gruppe informiert!\n" +
			"Nutze <code>/notify_disable</code> zum Deaktivieren.",
	})
}

func (p *Plugin) disableNotify(b *gotgbot.Bot, c plugin.GobotContext) error {
	enabled, err := p.notifyService.Enabled(c.EffectiveChat, c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error during enabled check")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if !enabled {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Du wirst in dieser Gruppe nicht über neue Erwähnungen informiert.", utils.DefaultSendOptions())
		return err
	}

	err = p.notifyService.Disable(c.EffectiveChat, c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error while disabling notifications")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Du wirst nicht mehr über neue Erwähnungen in dieser Gruppe informiert.",
	})
}
