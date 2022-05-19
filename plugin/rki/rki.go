package rki

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("rki")

const BaseUrl = "https://api.corona-zahlen.org"

type (
	Plugin struct {
		rkiService Service
	}

	Service interface {
		DelAGS(user *telebot.User) error
		GetAGS(user *telebot.User) (string, error)
		SetAGS(user *telebot.User, ags string) error
	}
)

func New(rkiService Service) *Plugin {
	return &Plugin{
		rkiService: rkiService,
	}
}

func (p *Plugin) Name() string {
	return "rki"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "rki",
			Description: "<Stadt> - COVID-19-Fälle in dieser deutschen Stadt",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/rki(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onNational,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/rki(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onDistrictSearch,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/rki_(\d+)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onDistrict,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/setrki_(\d+)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.setRkiAGS,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/myrki(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onMyRKI,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/delrki(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.delRKI,
		},
	}
}

func onNational(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	var response Nationwide

	url := fmt.Sprintf("%s/germany", BaseUrl)
	err := utils.GetRequest(
		url,
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("error getting RKI data")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var sb strings.Builder

	sb.WriteString("🇩🇪 <b>COVID-19-Übersicht lt. RKI:</b>\n")

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Million)\n",
			utils.FormatThousand(response.Cases),
			utils.FormatThousand(response.Delta.Cases),
			utils.FormatThousand(int(response.CasesPer100K*10)),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Fälle pro Woche:</b> %s\n",
			utils.FormatThousand(response.CasesPerWeek),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s (+ %s)\n",
			utils.FormatThousand(response.Recovered),
			utils.FormatThousand(response.Delta.Recovered),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s)\n",
			utils.FormatThousand(response.Deaths),
			utils.FormatThousand(response.Delta.Deaths),
		),
	)

	weekIncidence := fmt.Sprintf("%.2f", response.WeekIncidence)
	weekIncidence = strings.ReplaceAll(weekIncidence, ".", ",")
	sb.WriteString(
		fmt.Sprintf(
			"\n<b>7-Tage-Inzidenz:</b> %s\n",
			weekIncidence,
		),
	)

	if response.R.RValue4Days.Value > 0 {
		rValue4Days := fmt.Sprintf("%.2f", response.R.RValue4Days.Value)
		rValue4Days = strings.ReplaceAll(rValue4Days, ".", ",")
		sb.WriteString(
			fmt.Sprintf(
				"<b>4-Tage R-Wert:</b>: %s (vom %s)\n",
				rValue4Days,
				response.R.RValue4Days.Date.Format("02.01.2006"),
			),
		)
	}

	if response.R.RValue7Days.Value > 0 {
		rValue7Days := fmt.Sprintf("%.2f", response.R.RValue7Days.Value)
		rValue7Days = strings.ReplaceAll(rValue7Days, ".", ",")
		sb.WriteString(
			fmt.Sprintf(
				"<b>7-Tage R-Wert:</b>: %s (vom %s)\n",
				rValue7Days,
				response.R.RValue7Days.Date.Format("02.01.2006"),
			),
		)
	}

	timezone := utils.GermanTimezone()
	sb.WriteString(
		fmt.Sprintf(
			"\n<i>Zuletzt aktualisiert: %s</i>",
			response.Meta.LastUpdate.In(timezone).Format("02.01.2006"),
		),
	)

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func onDistrictSearch(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	var response DistrictResponse

	url := fmt.Sprintf("%s/districts", BaseUrl)
	err := utils.GetRequest(
		url,
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("error getting RKI data")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var foundDistricts []District
	for _, district := range response.Districts {
		if JaroWinklerDistance(strings.ToLower(district.Name), strings.ToLower(c.Matches[1])) > 0.9 {
			foundDistricts = append(foundDistricts, district)
		}
	}

	if len(foundDistricts) == 0 {
		return c.Reply("❌ Keine Stadt gefunden.", utils.DefaultSendOptions)
	}

	var sb strings.Builder
	for i, district := range foundDistricts {
		if i > 4 {
			break
		}
		sb.WriteString(
			fmt.Sprintf(
				"/rki_%s - <strong>%s (%s, %s)</strong>\n",
				district.Ags,
				utils.Escape(district.Name),
				utils.Escape(district.County),
				utils.Escape(district.State),
			),
		)
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func districtText(ags string) string {
	url := fmt.Sprintf("%s/districts/%s", BaseUrl, ags)
	var response DistrictResponse
	err := utils.GetRequest(
		url,
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("error getting RKI data")
		return fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid))
	}

	if len(response.Districts) == 0 {
		return "❌ Stadt nicht gefunden."
	}

	district := response.Districts[ags]

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Übersicht für %s (%s, %s) lt. RKI:</b>\n",
			utils.Escape(district.Name),
			utils.Escape(district.County),
			utils.Escape(district.State),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Gesamt:</b> %s (+ %s) (%s pro Million)\n",
			utils.FormatThousand(district.Cases),
			utils.FormatThousand(district.Delta.Cases),
			utils.FormatThousand(int(district.CasesPer100K*10)),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Fälle pro Woche:</b> %s\n",
			utils.FormatThousand(district.CasesPerWeek),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Genesen:</b> %s (+ %s)\n",
			utils.FormatThousand(district.Recovered),
			utils.FormatThousand(district.Delta.Recovered),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<b>Todesfälle:</b> %s (+ %s)\n",
			utils.FormatThousand(district.Deaths),
			utils.FormatThousand(district.Delta.Deaths),
		),
	)

	weekIncidence := fmt.Sprintf("%.2f", district.WeekIncidence)
	weekIncidence = strings.ReplaceAll(weekIncidence, ".", ",")
	sb.WriteString(
		fmt.Sprintf(
			"\n<b>7-Tage-Inzidenz:</b> %s\n",
			weekIncidence,
		),
	)

	timezone := utils.GermanTimezone()
	sb.WriteString(
		fmt.Sprintf(
			"\n<i>Zuletzt aktualisiert: %s</i>\n",
			response.Meta.LastUpdate.In(timezone).Format("02.01.2006"),
		),
	)

	sb.WriteString(
		fmt.Sprintf(
			"<i>Als Heimatstadt setzen: /setrki_%s</i>",
			utils.Escape(district.Ags),
		),
	)

	return sb.String()
}

func onDistrict(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	return c.Reply(districtText(c.Matches[1]), utils.DefaultSendOptions)
}

func (p *Plugin) setRkiAGS(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	ags := c.Matches[1]

	if len(ags) > 8 {
		return c.Reply("❌ Gemeindeschlüssel muss kleiner als 8 Zeichen lang sein.\nSuche mit <code>/rki STADT</code>.",
			utils.DefaultSendOptions)
	}

	var response DistrictResponse
	url := fmt.Sprintf("%s/districts/%s", BaseUrl, ags)
	err := utils.GetRequest(
		url,
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("error getting RKI data")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.Districts) == 0 {
		return c.Reply("❌ Stadt nicht gefunden.", utils.DefaultSendOptions)
	}

	err = p.rkiService.SetAGS(c.Sender(), ags)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.Sender().ID).
			Str("ags", ags).
			Msg("error while saving AGS")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(xid.New().String())),
			utils.DefaultSendOptions)
	}

	return c.Reply("✅ Du kannst jetzt /myrki nutzen.", utils.DefaultSendOptions)
}

func (p *Plugin) onMyRKI(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	ags, err := p.rkiService.GetAGS(c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.Sender().ID).
			Msg("error while getting AGS")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if ags == "" {
		return c.Reply("❌ Du hast keinen Gemeindeschlüssel gespeichert."+
			"\nSuche mit <code>/rki STADT</code> und setze ihn mit <code>/setrki_AGS</code>.",
			utils.DefaultSendOptions)
	}

	return c.Reply(districtText(ags), utils.DefaultSendOptions)
}

func (p *Plugin) delRKI(c plugin.GobotContext) error {
	err := p.rkiService.DelAGS(c.Sender())
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.Sender().ID).
			Msg("error while deleting AGS")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	return c.Reply("✅", utils.DefaultSendOptions)
}
