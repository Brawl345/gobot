package home

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("home")

type (
	Plugin struct {
		geocodingService model.GeocodingService
		homeService      model.HomeService
	}
)

func New(geocodingService model.GeocodingService, homeService model.HomeService) *Plugin {
	return &Plugin{
		geocodingService: geocodingService,
		homeService:      homeService,
	}
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "home",
			Description: "<Ort> - Heimatort setzen",
		},
		{
			Command:     "home_delete",
			Description: "Heimatort löschen",
		},
	}
}

func (p *Plugin) Name() string {
	return "home"
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/home(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onGetHome,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/home(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onHomeSet,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/home_delete(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onDeleteHome,
		},
	}
}

func (p *Plugin) onGetHome(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionFindLocation, nil)
	venue, err := p.homeService.GetHome(c.EffectiveUser)
	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			_, err = c.EffectiveMessage.Reply(b, "🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error getting home")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	_, err = b.SendVenue(c.EffectiveChat.Id, venue.Location.Latitude, venue.Location.Longitude, venue.Title, venue.Address, &gotgbot.SendVenueOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		DisableNotification: true,
	})
	return err
}

func (p *Plugin) onHomeSet(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionFindLocation, nil)

	venue, err := p.geocodingService.Geocode(c.Matches[1])

	if err != nil {
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Es wurde kein Ort gefunden.", utils.DefaultSendOptions())
			return err
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting location")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	err = p.homeService.SetHome(c.EffectiveUser, &venue)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error setting home")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}
	venue.Title = "✅ Wohnort festgelegt"

	_, err = b.SendVenue(c.EffectiveChat.Id, venue.Location.Latitude, venue.Location.Longitude, venue.Title, venue.Address, &gotgbot.SendVenueOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		DisableNotification: true,
	})
	return err
}

func (p *Plugin) onDeleteHome(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.homeService.DeleteHome(c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error deleting home")
		_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}
	_, err = c.EffectiveMessage.Reply(b, "✅ Wohnort gelöscht", utils.DefaultSendOptions())
	return err
}
