package home

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
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
	_ = c.Notify(telebot.FindingLocation)
	venue, err := p.homeService.GetHome(c.EffectiveUser)
	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			return c.Reply("🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error getting home")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	_, err := c.EffectiveMessage.Reply(b, &venue, utils.DefaultSendOptions)
	return err
}

func (p *Plugin) onHomeSet(b *gotgbot.Bot, c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)

	venue, err := p.geocodingService.Geocode(c.Matches[1])

	if err != nil {
		if errors.Is(err, model.ErrAddressNotFound) {
			_, err := c.EffectiveMessage.Reply(b, "❌ Es wurde kein Ort gefunden.", utils.DefaultSendOptions)
			return err
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting location")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	err = p.homeService.SetHome(c.EffectiveUser, &venue)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.EffectiveUser.Id).
			Str("guid", guid).
			Msg("error setting home")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}
	venue.Title = "✅ Wohnort festgelegt"
	_, err := c.EffectiveMessage.Reply(b, &venue, utils.DefaultSendOptions)
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
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}
	_, err := c.EffectiveMessage.Reply(b, "✅ Wohnort gelöscht", utils.DefaultSendOptions)
	return err
}
