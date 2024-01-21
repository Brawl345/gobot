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
	"gopkg.in/telebot.v3"
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

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "home",
			Description: "<Ort> - Heimatort setzen",
		},
		{
			Text:        "home_delete",
			Description: "Heimatort löschen",
		},
	}
}

func (p *Plugin) Name() string {
	return "home"
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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

func (p *Plugin) onGetHome(c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)
	venue, err := p.homeService.GetHome(c.Sender())
	if err != nil {
		if errors.Is(err, model.ErrHomeAddressNotSet) {
			return c.Reply("🏠 Dein Heimatort wurde noch nicht gesetzt.\n"+
				"Setze ihn mit <code>/home ORT</code>", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error getting home")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	return c.Reply(&venue, utils.DefaultSendOptions)
}

func (p *Plugin) onHomeSet(c plugin.GobotContext) error {
	_ = c.Notify(telebot.FindingLocation)

	venue, err := p.geocodingService.Geocode(c.Matches[1])

	if err != nil {
		if errors.Is(err, model.ErrAddressNotFound) {
			return c.Reply("❌ Es wurde kein Ort gefunden.", utils.DefaultSendOptions)
		}
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Msg("error getting location")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	err = p.homeService.SetHome(c.Sender(), &venue)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error setting home")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}
	venue.Title = "✅ Wohnort festgelegt"
	return c.Reply(&venue, utils.DefaultSendOptions)
}

func (p *Plugin) onDeleteHome(c plugin.GobotContext) error {
	err := p.homeService.DeleteHome(c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Int64("user_id", c.Sender().ID).
			Str("guid", guid).
			Msg("error deleting home")
		return c.Reply(fmt.Sprintf("❌ Ein Fehler ist aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}
	return c.Reply("✅ Wohnort gelöscht", utils.DefaultSendOptions)
}
