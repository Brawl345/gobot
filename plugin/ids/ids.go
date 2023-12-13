package ids

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

var log = logger.New("ids")

type (
	Plugin struct {
		idsService Service
	}

	Service interface {
		GetAllUsersInChat(chat *telebot.Chat) ([]model.User, error)
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "ids",
			Description: "Zeigt die IDs der User in diesem Chat an",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/ids(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onIds,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) onIds(c plugin.GobotContext) error {
	users, err := p.idsService.GetAllUsersInChat(c.Message().Chat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to get all users in chat")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	memberCount, err := c.Bot().Len(c.Message().Chat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to count members in chat")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	adminsAndCreators, err := c.Bot().AdminsOf(c.Message().Chat)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to get admins and creators in chat")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}

	var admins []int64
	var creator int64
	for _, u := range adminsAndCreators {
		if u.Role == telebot.Creator {
			creator = u.User.ID
		} else if u.Role == telebot.Administrator {
			admins = append(admins, u.User.ID)
		}
	}

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"👥 <b>%s</b> <code>%d</code>\n",
			utils.Escape(c.Chat().Title),
			c.Chat().ID,
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

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
