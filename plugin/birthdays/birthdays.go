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
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("birthdays")

type (
	Plugin struct {
		birthdayService Service
	}

	Service interface {
		BirthdayNotificationsEnabled(chat *telebot.Chat) (bool, error)
		DeleteBirthday(user *telebot.User) error
		Birthdays(chat *telebot.Chat) ([]model.User, error)
		DisableBirthdayNotifications(chat *telebot.Chat) error
		EnableBirthdayNotifications(chat *telebot.Chat) error
		SetBirthday(user *telebot.User, birthday time.Time) error
		TodaysBirthdays() (map[int64][]model.User, error)
	}
)

func New(bot *telebot.Bot, birthdayService Service) *Plugin {
	p := &Plugin{
		birthdayService: birthdayService,
	}
	p.scheduleNewRun(bot)
	return p
}

func (p *Plugin) Name() string {
	return "birthdays"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "bday",
			Description: "<TT.MM.JJJJ> - Geburtstag setzen",
		},
		{
			Text:        "bday_delete",
			Description: "Geburtstag löschen",
		},
		{
			Text:        "bdays",
			Description: "Geburtstage anzeigen, falls Benachrichtigungen aktiv",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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

func (p *Plugin) scheduleNewRun(bot *telebot.Bot) {
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

func (p *Plugin) onNewDay(bot *telebot.Bot) {
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
			_, err := bot.Send(telebot.ChatID(chatID), text, utils.DefaultSendOptions)
			if err != nil {
				log.Err(err).Msg("Failed to send birthday message")
			}
		}
	}
}

func (p *Plugin) onSetBirthday(c plugin.GobotContext) error {
	birthday, err := time.Parse("02.01.2006", c.Matches[1])

	if err != nil {
		return c.Reply("❌ <b>Ungültiges Datum.</b> Bitte im Format <code>TT.MM.JJJJ</code> eingeben.",
			utils.DefaultSendOptions)
	}

	if birthday.IsZero() {
		return c.Reply("❌ Ungültiges Datum.", utils.DefaultSendOptions)
	}

	if birthday.After(time.Now()) {
		return c.Reply("❌ Ich glaube nicht, dass du erst noch geboren werden musst.", utils.DefaultSendOptions)
	}

	if birthday.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.Local)) {
		return c.Reply("❌ Ich glaube nicht, dass du so alt bist.", utils.DefaultSendOptions)
	}

	err = p.birthdayService.SetBirthday(c.Sender(), birthday)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Time("birthday", birthday).
			Msg("Failed to set birthday")
	}

	return c.Reply("✅ <b>Dein Geburtstag wurde gespeichert.</b>", utils.DefaultSendOptions)
}

func (p *Plugin) onDeleteBirthday(c plugin.GobotContext) error {
	err := p.birthdayService.DeleteBirthday(c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to delete birthday")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("✅ <b>Dein Geburtstag wurde gelöscht.</b>\nTja, ich schätze du alterst nicht mehr.", utils.DefaultSendOptions)
}

func (p *Plugin) onEnableBirthdayNotifications(c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	if enabled {
		return c.Reply("💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe schon aktiv.",
			utils.DefaultSendOptions)
	}

	err = p.birthdayService.EnableBirthdayNotifications(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to enable birthday notifications")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("✅ Geburtstagsbenachrichtigungen wurden aktiviert.", utils.DefaultSendOptions)
}

func (p *Plugin) onDisableBirthdayNotifications(c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	if !enabled {
		return c.Reply("💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe nicht aktiv.",
			utils.DefaultSendOptions)
	}

	err = p.birthdayService.DisableBirthdayNotifications(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to disable birthday notifications")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("✅ Geburtstagsbenachrichtigungen wurden deaktiviert.", utils.DefaultSendOptions)
}

func (p *Plugin) listBirthdays(c plugin.GobotContext) error {
	enabled, err := p.birthdayService.BirthdayNotificationsEnabled(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to get birthday notification state")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	if !enabled {
		return c.Reply("💡 Geburtsagsbenachrichtigungen sind in dieser Gruppe nicht aktiv, daher werden keine Geburtstage gelistet.",
			utils.DefaultSendOptions)
	}

	users, err := p.birthdayService.Birthdays(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Int64("chat_id", c.Chat().ID).
			Str("guid", guid).
			Msg("Failed to get birthdays")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	if len(users) == 0 {
		return c.Reply("💡 Es wurden noch keine Geburtstage eingespeichert.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>🎂 Geburtstage in %s:</b>\n",
			utils.Escape(c.Chat().Title),
		),
	)

	for _, user := range users {
		sb.WriteString(
			fmt.Sprintf(
				"<b>%s:</b> %s\n",
				utils.Escape(user.GetFullName()),
				user.Birthday.Time.Format("02.01.2006"),
			),
		)
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)

}
