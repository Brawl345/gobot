package reminders

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
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

var log = logger.New("reminders")

type (
	Plugin struct {
		reminderService Service
	}

	Service interface {
		DeleteReminder(chat *gotgbot.Chat, user *gotgbot.User, id string) error
		DeleteReminderByID(id int64) error
		GetReminderByID(id int64) (model.Reminder, error)
		GetAllReminders() ([]model.Reminder, error)
		GetReminders(chat *gotgbot.Chat, user *gotgbot.User) ([]model.Reminder, error)
		SaveReminder(chat *gotgbot.Chat, user *gotgbot.User, remindAt time.Time, text string) (int64, error)
	}
)

func New(bot *gotgbot.Bot, service Service) *Plugin {
	reminders, err := service.GetAllReminders()
	if err != nil {
		log.Err(err).
			Msg("Failed to get all reminders")
	}

	p := &Plugin{
		reminderService: service,
	}

	for _, reminder := range reminders {
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

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "remind",
			Description: "<Zeit> <Text> - Erinnerung speichern. Unterstützt absolute und relative Zeitangaben",
		},
		{
			Command:     "reminders",
			Description: "Alle Erinnerungen anzeigen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func (p *Plugin) onAddDateTimeReminder(b *gotgbot.Bot, c plugin.GobotContext) error {
	text := c.Matches[5]
	day, err := strconv.ParseInt(c.Matches[1], 10, 32)
	if err != nil {
		log.Err(err).
			Str("day", c.Matches[1]).
			Msg("Failed to parse hour")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	month, err := strconv.ParseInt(c.Matches[2], 10, 32)
	if err != nil {
		log.Err(err).
			Str("month", c.Matches[2]).
			Msg("Failed to parse minutes")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	hour, err := strconv.ParseInt(c.Matches[3], 10, 32)
	if err != nil {
		log.Err(err).
			Str("hour", c.Matches[1]).
			Msg("Failed to parse hour")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	minute, err := strconv.ParseInt(c.Matches[4], 10, 32)
	if err != nil {
		log.Err(err).
			Str("minute", c.Matches[2]).
			Msg("Failed to parse minutes")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	_, err = time.Parse("02.01.2006 15:04",
		fmt.Sprintf("%02d.%02d.%d %02d:%02d", day, month, time.Now().Year(), hour, minute),
	)
	if err != nil {
		log.Err(err).
			Int64("day", day).
			Int64("month", month).
			Msg("Unsupported unit")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib ein gültiges Datum an.", utils.DefaultSendOptions())
		return err
	}

	now := time.Now()
	remindTime := time.Date(
		now.Year(),
		time.Month(month),
		int(day),
		int(hour),
		int(minute),
		0,
		0,
		time.Local,
	)

	if remindTime.Before(now) {
		remindTime = remindTime.AddDate(1, 0, 0)
	}

	id, err := p.reminderService.SaveReminder(c.EffectiveChat, c.EffectiveUser, remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(b, id)
	})

	_, err = c.EffectiveMessage.ReplyMessage(b,
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onAddTimeReminder(b *gotgbot.Bot, c plugin.GobotContext) error {
	text := c.Matches[3]
	hour, err := strconv.ParseInt(c.Matches[1], 10, 32)
	if err != nil {
		log.Err(err).
			Str("hour", c.Matches[1]).
			Msg("Failed to parse hour")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	minute, err := strconv.ParseInt(c.Matches[2], 10, 32)
	if err != nil {
		log.Err(err).
			Str("minute", c.Matches[2]).
			Msg("Failed to parse minutes")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	_, err = time.Parse("15:04", fmt.Sprintf("%02d:%02d", hour, minute))
	if err != nil {
		log.Err(err).
			Int64("hour", hour).
			Int64("minute", minute).
			Msg("Unsupported unit")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte gib eine gültige Uhrzeit an.", utils.DefaultSendOptions())
		return err
	}

	now := time.Now()
	remindTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		int(hour),
		int(minute),
		0,
		0,
		time.Local,
	)

	if remindTime.Before(now) {
		remindTime = remindTime.AddDate(0, 0, 1)
	}

	id, err := p.reminderService.SaveReminder(c.EffectiveChat, c.EffectiveUser, remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(b, id)
	})

	_, err = c.EffectiveMessage.ReplyMessage(b,
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onAddDeltaReminder(b *gotgbot.Bot, c plugin.GobotContext) error {
	dur, err := strconv.ParseInt(c.Matches[1], 10, 64)
	if err != nil {
		log.Err(err).
			Str("duration", c.Matches[1]).
			Msg("Failed to parse amount")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte wähle eine kürzere Dauer.", utils.DefaultSendOptions())
		return err
	}
	unit := c.Matches[2]
	text := c.Matches[3]

	var unitDuration time.Duration
	switch unit {
	case "h":
		unitDuration = time.Hour
	case "m":
		unitDuration = time.Minute
	case "s":
		unitDuration = time.Second
	default:
		log.Err(err).
			Str("unit", unit).
			Msg("Unsupported unit")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte wähle als Zeitangabe entweder 's', 'm' oder 'h'.", utils.DefaultSendOptions())
		return err
	}

	// Guard against int64 overflow when scaling dur to a Duration, which would
	// wrap to a negative offset and fire immediately.
	if dur > int64(math.MaxInt64)/int64(unitDuration) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte wähle eine kürzere Dauer.", utils.DefaultSendOptions())
		return err
	}

	remindTime := time.Now().Add(time.Duration(dur) * unitDuration)

	if remindTime.After(time.Now().AddDate(1, 0, 0)) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Bitte wähle eine kürzere Dauer.", utils.DefaultSendOptions())
		return err
	}

	id, err := p.reminderService.SaveReminder(c.EffectiveChat, c.EffectiveUser, remindTime, text)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to save reminder")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	time.AfterFunc(time.Until(remindTime), func() {
		p.sendReminder(b, id)
	})

	_, err = c.EffectiveMessage.ReplyMessage(b,
		fmt.Sprintf("🕒 Erinnerung eingestellt für den <b>%s</b>.",
			remindTime.Format("02.01.2006 um 15:04:05 Uhr"),
		),
		utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onDeleteReminder(b *gotgbot.Bot, c plugin.GobotContext) error {
	id := c.Matches[1]
	err := p.reminderService.DeleteReminder(c.EffectiveChat, c.EffectiveUser, id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Diese Erinnerung existiert nicht.", &gotgbot.SendMessageOpts{
				ReplyParameters: &gotgbot.ReplyParameters{
					AllowSendingWithoutReply: true,
				},
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
				DisableNotification: true,
				ParseMode:           gotgbot.ParseModeHTML,
			})
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to delete reminder")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Erinnerung gelöscht.",
	})
}

func (p *Plugin) onGetReminders(b *gotgbot.Bot, c plugin.GobotContext) error {
	reminders, err := p.reminderService.GetReminders(c.EffectiveChat, c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Failed to get reminders")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Es ist ein Fehler aufgetreten.", utils.DefaultSendOptions())
		return err
	}

	if len(reminders) == 0 {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Es wurden noch keine Erinnerungen eingespeichert.", utils.DefaultSendOptions())
		return err
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

	_, err = c.EffectiveMessage.ReplyMessage(b, sb.String(), utils.DefaultSendOptions())
	return err

}

func (p *Plugin) sendReminder(bot *gotgbot.Bot, id int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Int64("id", id).Msg("Recovered from panic in sendReminder")
		}
	}()

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

	_, err = bot.SendMessage(
		recipient,
		sb.String(),
		utils.DefaultSendOptions(),
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
