package creds

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("creds")

type Plugin struct {
	credentialService model.CredentialService
}

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (*Plugin) Name() string {
	return "creds"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil // Because it's a superuser plugin
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnGet,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds_add(?:@%s)? ([^\s]+) ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.OnAdd,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds_del(?:@%s)? ([^\s]+)$`, botInfo.Username)),
			HandlerFunc: p.OnDelete,
			AdminOnly:   true,
		},
		&plugin.CallbackHandler{
			HandlerFunc: p.OnHide,
			Trigger:     regexp.MustCompile(`^creds_hide$`),
			AdminOnly:   true,
		},
	}
}

func (p *Plugin) OnGet(b *gotgbot.Bot, c plugin.GobotContext) error {
	if tgUtils.FromGroup(c.EffectiveMessage) {
		return nil
	}

	creds := p.credentialService.GetAllCredentials()

	if len(creds) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "<i>Noch keine Schlüssel eingetragen</i>", utils.DefaultSendOptions())
		return err
	}

	// Sort creds alphabetically
	keys := make([]string, 0, len(creds))
	for k := range creds {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder

	for _, key := range keys {
		if len(creds[key]) > 300 {
			sb.WriteString(fmt.Sprintf("<b>%s</b>:\n<code>%s...</code>\n", key, creds[key][:297]))
		} else {
			sb.WriteString(fmt.Sprintf("<b>%s</b>:\n<code>%s</code>\n", key, creds[key]))
		}
	}

	_, err := c.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeHTML,
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "Verbergen",
						CallbackData: "creds_hide",
					},
				},
			},
		},
	})
	return err
}

func (p *Plugin) OnAdd(b *gotgbot.Bot, c plugin.GobotContext) error {
	if tgUtils.FromGroup(c.EffectiveMessage) {
		return nil
	}

	key := c.Matches[1]
	value := c.Matches[2]

	err := p.credentialService.SetKey(key, value)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Error adding key")
		_, err := c.EffectiveMessage.Reply(b, "❌ Fehler beim Speichern des Schlüssels", utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, "✅ Schlüssel gespeichert.", utils.DefaultSendOptions())
	return err
}

func (p *Plugin) OnDelete(b *gotgbot.Bot, c plugin.GobotContext) error {
	if tgUtils.FromGroup(c.EffectiveMessage) {
		return nil
	}

	key := c.Matches[1]
	err := p.credentialService.DeleteKey(key)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Error deleting key")

		_, err := c.EffectiveMessage.Reply(
			b,
			fmt.Sprintf("❌ Fehler beim Löschen des Schlüssels.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions(),
		)
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, "✅ Schlüssel gelöscht.", utils.DefaultSendOptions())
	return err
}

func (p *Plugin) OnHide(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, err := b.DeleteMessage(c.EffectiveChat.Id, c.EffectiveMessage.MessageId, nil)
	if err != nil {
		log.Err(err).Send()
	}
	_, err = c.CallbackQuery.Answer(b, nil)
	return err
}
