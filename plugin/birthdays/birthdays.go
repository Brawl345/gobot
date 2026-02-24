package birthdays

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("birthdays")

type (
	Plugin struct {
		birthdayService Service
	}

	Service interface {
		BirthdayNotificationsEnabled(chat *gotgbot.Chat) (bool, error)
		DeleteBirthday(user *gotgbot.User) error
		Birthdays(chat *gotgbot.Chat) ([]model.User, error)
		DisableBirthdayNotifications(chat *gotgbot.Chat) error
		EnableBirthdayNotifications(chat *gotgbot.Chat) error
		SetBirthday(user *gotgbot.User, birthday time.Time) error
		TodaysBirthdays() (map[int64][]model.User, error)
	}
)

func New(bot *gotgbot.Bot, birthdayService Service) *Plugin {
	p := &Plugin{
		birthdayService: birthdayService,
	}
	p.scheduleNewRun(bot)
	return p
}

func (p *Plugin) Name() string {
	return "birthdays"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "bday",
			Description: "<TT.MM.JJJJ> - Geburtstag setzen",
		},
		{
			Command:     "bday_delete",
			Description: "Geburtstag löschen",
		},
		{
			Command:     "bdays",
			Description: "Geburtstage anzeigen, falls Benachrichtigungen aktiv",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/b(?:irth)?day(?:@%s)? (\d{2}\.\d{2}\.\d{4})$`, botInfo.Username)),
			HandlerFunc: p.onSetBirthday,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/b(?:irth)?day_delete(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onDeleteBirthday,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/b(?:irth)?days?_enable(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onEnableBirthdayNotifications,
			GroupOnly:   true,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/b(?:irth)?days?_disable(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onDisableBirthdayNotifications,
			GroupOnly:   true,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/b(?:irth)?days(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.listBirthdays,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) scheduleNewRun(bot *gotgbot.Bot) {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	midnight = midnight.AddDate(0, 0, 1)
	untilMidnight := time.Until(midnight)
	time.AfterFunc(untilMidnight, func() {
		p.onNewDay(bot)
	})
	log.Debug().
		Msgf("Scheduled new run at %s", time.Now().Add(untilMidnight).Format("2006-01-02 15:04:05"))
}

func (p *Plugin) onNewDay(bot *gotgbot.Bot) {
	log.Debug().Msg("Checking for birthdays")
	defer p.scheduleNewRun(bot)

	birthdayList, err := p.birthdayService.TodaysBirthdays()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get todays birthdays")
		return
	}

	for chatID, list := range birthdayList {
		for _, user := range list {
			age := time.Now().Year() - user.Birthday.Time.Year()
			text := fmt.Sprintf("🎂🍰🎈<b>%s hat heute Geburtstag und wird %d!</b>🎉🎁🕯\nAlles Gute!",
				utils.Escape(user.FirstName), age)
			_, err := bot.SendMessage(chatID, text, utils.DefaultSendOptions())
			if err != nil {
				log.Err(err).Msg("Failed to send birthday message")
			}
		}
	}
}

func (p *Plugin) onSetBirthday(b *gotgbot.Bot, c plugin.GobotContext) error {
	birthday, err := time.Parse("02.01.2006", c.Matches[1])

	if err != nil {
		_, err := c.EffectiveMessage.ReplyMessage(b,
			"❌ <b>Ungültiges Datum.</b> Bitte im Format <code>TT.MM.JJJJ</code> eingeben.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	if birthday.IsZero() {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Ungültiges Datum.", utils.DefaultSendOptions())
		return err
	}

	if birthday.After(time.Now()) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Ich glaube nicht, dass du erst noch geboren werden musst.", utils.DefaultSendOptions())
		return err
	}

	if birthday.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.Local)) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Ich glaube nicht, dass du so alt bist.", utils.DefaultSendOptions())
		return err
	}

	err = p.birthdayService.SetBirthday(c.EffectiveUser, birthday)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Time("birthday", birthday).
			Msg("Failed to set birthday")
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍",
		&tgUtils.ReactionFallbackOpts{
			Fallback: "✅ <b>Dein Geburtstag wurde gespeichert.</b>",
		},
	)
}

func (p *Plugin) onDeleteBirthday(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.birthdayService.DeleteBirthday(c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to delete birthday")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍",
		&tgUtils.ReactionFallbackOpts{
			Fallback: "✅ <b>Dein Geburtstag wurde gelöscht.</b>\nTja, ich schätze du alterst nicht mehr.",
		},
	)
}

func (p *Plugin) onEnableBirthdayNotifications(b *gotgbot.Bot, c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}
	if enabled {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe schon aktiv.",
			utils.DefaultSendOptions())
		return err
	}

	err = p.birthdayService.EnableBirthdayNotifications(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to enable birthday notifications")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍",
		&tgUtils.ReactionFallbackOpts{
			Fallback: "✅ Geburtstagsbenachrichtigungen wurden aktiviert.",
		},
	)
}

func (p *Plugin) onDisableBirthdayNotifications(b *gotgbot.Bot, c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}
	if !enabled {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe nicht aktiv.",
			utils.DefaultSendOptions())
		return err
	}

	err = p.birthdayService.DisableBirthdayNotifications(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to disable birthday notifications")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍",
		&tgUtils.ReactionFallbackOpts{
			Fallback: "✅ Geburtstagsbenachrichtigungen wurden deaktiviert.",
		},
	)
}

func (p *Plugin) listBirthdays(b *gotgbot.Bot, c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}
	if !enabled {
		_, err := c.EffectiveMessage.ReplyMessage(b,
			"💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe nicht aktiv, daher werden keine Geburtstage gelistet.",
			utils.DefaultSendOptions())
		return err
	}

	users, err := p.birthdayService.Birthdays(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Str("guid", guid).
			Msg("Failed to get birthdays")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(users) == 0 {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Es wurden noch keine Geburtstage eingespeichert.", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>🎂 Geburtstage in %s:</b>\n",
			utils.Escape(c.EffectiveChat.Title),
		),
	)

	for _, user := range users {
		now := time.Now()
		age := now.Year() - user.Birthday.Time.Year()
		if now.YearDay() < user.Birthday.Time.YearDay() {
			age--
		}
		sb.WriteString(
			fmt.Sprintf(
				"<b>%s:</b> %s (%d)\n",
				utils.Escape(user.GetFullName()),
				user.Birthday.Time.Format("02.01.2006"),
				age,
			),
		)
	}

	_, err = c.EffectiveMessage.ReplyMessage(b, sb.String(), utils.DefaultSendOptions())
	return err

}
