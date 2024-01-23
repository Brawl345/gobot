package ids

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
	"golang.org/x/exp/slices"
)

var log = logger.New("ids")

type (
	Plugin struct {
		idsService Service
	}

	Service interface {
		GetAllUsersInChat(chat *gotgbot.Chat) ([]model.User, error)
	}
)

func New(idsService Service) *Plugin {
	return &Plugin{
		idsService: idsService,
	}
}

func (p *Plugin) Name() string {
	return "ids"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "ids",
			Description: "Zeigt die IDs der User in diesem Chat an",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/ids(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onIds,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) onIds(b *gotgbot.Bot, c plugin.GobotContext) error {
	users, err := p.idsService.GetAllUsersInChat(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to get all users in chat")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	memberCount, err := c.EffectiveChat.GetMemberCount(b, nil)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to count members in chat")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	adminsAndCreators, err := c.EffectiveChat.GetAdministrators(b, nil)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to get admins and creators in chat")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
		return err
	}

	var admins []int64
	var creator int64
	for _, u := range adminsAndCreators {
		if u.GetStatus() == utils.ChatMemberStatusCreator {
			creator = u.GetUser().Id
		} else if u.GetStatus() == utils.ChatMemberStatusAdministrator {
			admins = append(admins, u.GetUser().Id)
		}
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"👥 <b>%s</b> <code>%d</code>\n",
			utils.Escape(c.EffectiveChat.Title),
			c.EffectiveChat.Id,
		),
	)

	var plural string
	if memberCount > 1 {
		plural = "er"
	}
	sb.WriteString(
		fmt.Sprintf(
			"%d Mitglied%s\n",
			memberCount,
			plural,
		),
	)

	sb.WriteString("============================\n")

	for _, user := range users {
		sb.WriteString(
			fmt.Sprintf(
				"<b>%s</b> <code>%d</code>",
				utils.Escape(utils.FullName(user.FirstName, user.LastName.String)),
				user.ID,
			),
		)

		if slices.Contains(admins, user.ID) {
			sb.WriteString(" <i>Admin</i>")
		} else if user.ID == creator {
			sb.WriteString(" <i>Gründer</i>")
		}

		sb.WriteString("\n")
	}

	sb.WriteString("<i>(Bots sind nicht gelistet)</i>")

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}
