package reminders

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("reminders")

type (
	Plugin struct {
		reminderService Service
	}

	Service interface {
		DeleteReminder(chat *telebot.Chat, user *telebot.User, id string) error
		DeleteReminderByID(id int64) error
		GetReminderByID(id int64) (model.Reminder, error)
		GetAllReminders() ([]model.Reminder, error)
		GetReminders(chat *telebot.Chat, user *telebot.User) ([]model.Reminder, error)
		SaveReminder(chat *telebot.Chat, user *telebot.User, remindAt time.Time, text string) (int64, error)
	}
)

func New(bot *telebot.Bot, service Service) *Plugin {
	reminders, err := service.GetAllReminders()
	if err != nil {
		log.Err(err).
			Msg("Failed to get all reminders")
	}

	p := &Plugin{
		reminderService: service,
	}

	for _, reminder := range reminders {
		reminder := reminder
		if time.Now().After(reminder.Time) {
			p.sendReminder(bot, reminder.ID)
		} else {
			time.AfterFunc(time.Until(reminder.Time), func() {
				p.sendReminder(bot, reminder.ID)
			})
		}
	}

	return p
}

func (p *Plugin) Name() string {
	return "reminders"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "remind",
			Description: "<Zeit> <Text> - Erinnerung speichern. Unterstützt absolute und relative Zeitangaben",
		},
		{
			Text:        "reminders",
			Description: "Alle Erinnerungen anzeigen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/remind(?:@%s)? (\d+).(\d+). (\d+):(\d+) (.+)$`, botInfo.Username)),
			HandlerFunc: p.onAddDateTimeReminder,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/remind(?:@%s)? (\d+):(\d+) (.+)$`, botInfo.Username)),
			HandlerFunc: p.onAddTimeReminder,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/remind(?:@%s)? (\d+)(h|m|s) (.+)$`, botInfo.Username)),
			HandlerFunc: p.onAddDeltaReminder,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/remind_delete(?:@%s)? (\d+)$`, botInfo.Username)),
			HandlerFunc: p.onDeleteReminder,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/reminders(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onGetReminders,
		},
	}
}

func (p *Plugin) onAddDateTimeReminder(c plugin.GobotContext) error {
	text := c.Matches[5]
	day, err := strconv.ParseInt(c.Matches[1], 10, 32)
	if err != nil {
		log.Err(err).
			Str("day", c.Matches[1]).
			Msg("Failed to parse hour")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	month, err := strconv.ParseInt(c.Matches[2], 10, 32)
	if err != nil {
		log.Err(err).
			Str("month", c.Matches[2]).
			Msg("Failed to parse minutes")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	hour, err := strconv.ParseInt(c.Matches[3], 10, 32)
	if err != nil {
		log.Err(err).
			Str("hour", c.Matches[1]).
			Msg("Failed to parse hour")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	min, err := strconv.ParseInt(c.Matches[4], 10, 32)
	if err != nil {
		log.Err(err).
			Str("min", c.Matches[2]).
			Msg("Failed to parse minutes")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	_, err = time.Parse("02.01.2006 15:05",
		fmt.Sprintf("%02d.%02d.%d %02d:%02d", day, month, time.Now().Year(), hour, min),
	)
	if err != nil {
		log.Err(err).
			Int64("day", day).
			Int64("month", month).
			Msg("Unsupported unit")
		return c.Reply("❌ Bitte gib ein gültiges Datum an.", utils.DefaultSendOptions)
	}

	now := time.Now()
	remindTime := time.Date(
		now.Year(),
		time.Month(month),
		int(day),
		int(hour),
		int(min),
		0,
		0,
		time.Local,
	)

	if remindTime.Before(now) {
		remindTime = remindTime.AddDate(1, 0, 0)
	}

	id, err := p.reminderService.SaveReminder(c.Chat(), c.Sender(), remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(c.Bot(), id)
	})

	return c.Reply(
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions)
}

func (p *Plugin) onAddTimeReminder(c plugin.GobotContext) error {
	text := c.Matches[3]
	hour, err := strconv.ParseInt(c.Matches[1], 10, 32)
	if err != nil {
		log.Err(err).
			Str("hour", c.Matches[1]).
			Msg("Failed to parse hour")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	min, err := strconv.ParseInt(c.Matches[2], 10, 32)
	if err != nil {
		log.Err(err).
			Str("min", c.Matches[2]).
			Msg("Failed to parse minutes")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	_, err = time.Parse("15:04", fmt.Sprintf("%02d:%02d", hour, min))
	if err != nil {
		log.Err(err).
			Int64("hour", hour).
			Int64("min", min).
			Msg("Unsupported unit")
		return c.Reply("❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions)
	}

	now := time.Now()
	remindTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		int(hour),
		int(min),
		0,
		0,
		time.Local,
	)

	if remindTime.Before(now) {
		remindTime = remindTime.AddDate(0, 0, 1)
	}

	id, err := p.reminderService.SaveReminder(c.Chat(), c.Sender(), remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(c.Bot(), id)
	})

	return c.Reply(
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions)
}

func (p *Plugin) onAddDeltaReminder(c plugin.GobotContext) error {
	dur, err := strconv.ParseInt(c.Matches[1], 10, 64)
	if err != nil {
		log.Err(err).
			Str("duration", c.Matches[1]).
			Msg("Failed to parse amount")
		return c.Reply("❌ Bitte wähle eine kürzere Dauer.", utils.DefaultSendOptions)
	}
	unit := c.Matches[2]
	text := c.Matches[3]

	remindTime := time.Now()

	switch unit {
	case "h":
		remindTime = remindTime.Add(time.Duration(dur) * time.Hour)
	case "m":
		remindTime = remindTime.Add(time.Duration(dur) * time.Minute)
	case "s":
		remindTime = remindTime.Add(time.Duration(dur) * time.Second)
	default:
		log.Err(err).
			Str("unit", unit).
			Msg("Unsupported unit")
		return c.Reply("❌ Bitte wähle als Zeitangabe entweder 's', 'm' oder 'h'.", utils.DefaultSendOptions)
	}

	if remindTime.After(time.Now().AddDate(1, 0, 0)) {
		return c.Reply("❌ Bitte wähle eine kürzere Dauer.", utils.DefaultSendOptions)
	}

	id, err := p.reminderService.SaveReminder(c.Chat(), c.Sender(), remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(c.Bot(), id)
	})

	return c.Reply(
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions)
}

func (p *Plugin) onDeleteReminder(c plugin.GobotContext) error {
	id := c.Matches[1]
	err := p.reminderService.DeleteReminder(c.Chat(), c.Sender(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.Reply("❌ Diese Erinnerung existiert nicht.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to delete reminder")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	return c.Reply("✅ Erinnerung gelöscht.", utils.DefaultSendOptions)
}

func (p *Plugin) onGetReminders(c plugin.GobotContext) error {
	reminders, err := p.reminderService.GetReminders(c.Chat(), c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get reminders")
		return c.Reply("❌ Es ist ein Fehler aufgetreten.", utils.DefaultSendOptions)
	}

	if len(reminders) == 0 {
		return c.Reply("💡 Es wurden noch keine Erinnerungen eingespeichert.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	for _, reminder := range reminders {
		sb.WriteString(
			fmt.Sprintf(
				"<b>%d)</b> %s - <b>%s</b>\n",
				reminder.ID,
				reminder.Time.Format("02.01.2006, 15:04:05 Uhr"),
				utils.Escape(reminder.Text),
			),
		)
	}

	sb.WriteString("\n<i>Zum Entfernen einer Erinnerung: <code>/remind_delete ID</code></i>")

	return c.Reply(sb.String(), utils.DefaultSendOptions)

}

func (p *Plugin) sendReminder(bot *telebot.Bot, id int64) {
	log.Debug().
		Int64("id", id).
		Msg("Sending reminder")

	reminder, err := p.reminderService.GetReminderByID(id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			log.Debug().
				Int64("id", id).
				Msg("Reminder not found, probably deleted")
			return
		}
		log.Err(err).
			Int64("id", id).
			Msg("Failed to get reminder")
		return
	}

	var sb strings.Builder
	recipient := reminder.UserID

	sb.WriteString("🔔🔔🔔 ")
	if reminder.ChatID.Valid {
		recipient = reminder.ChatID.Int64
		if reminder.Username != "" {
			sb.WriteString(
				fmt.Sprintf(
					"<b>@%s</b> ",
					utils.Escape(reminder.Username),
				),
			)
		}
	}
	sb.WriteString("<b>ERINNERUNG:</b>\n")
	sb.WriteString(utils.Escape(reminder.Text))

	_, err = bot.Send(
		telebot.ChatID(recipient),
		sb.String(),
		utils.DefaultSendOptions,
	)

	if err != nil {
		log.Err(err).
			Int64("id", id).
			Msg("Failed to send reminder")
		return
	}

	err = p.reminderService.DeleteReminderByID(id)
	if err != nil {
		log.Err(err).
			Int64("id", id).
			Msg("Failed to delete reminder")
	}
}
