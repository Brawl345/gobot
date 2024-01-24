package rki

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

var log = logger.New("rki")

const BaseUrl = "https://api.corona-zahlen.org"

type (
	Plugin struct {
		rkiService Service
	}

	Service interface {
		DelAGS(user *gotgbot.User) error
		GetAGS(user *gotgbot.User) (string, error)
		SetAGS(user *gotgbot.User, ags string) error
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

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "rki",
			Description: "<Stadt> - COVID-19-Fälle in dieser deutschen Stadt",
		},
		{
			Command:     "myrki",
			Description: "COVID-19-Fälle in deinem Heimatort",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func onNational(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	var response Nationwide

	url := fmt.Sprintf("%s/germany", BaseUrl)
	err := httpUtils.GetRequest(
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
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
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

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}

func onDistrictSearch(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	var response DistrictResponse

	url := fmt.Sprintf("%s/districts", BaseUrl)
	err := httpUtils.GetRequest(
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
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	var foundDistricts []District
	for _, district := range response.Districts {
		if JaroWinklerDistance(strings.ToLower(district.Name), strings.ToLower(c.Matches[1])) > 0.9 {
			foundDistricts = append(foundDistricts, district)
		}
	}

	if len(foundDistricts) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Stadt gefunden.", utils.DefaultSendOptions())
		return err
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

	_, err = c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}

func districtText(ags string) string {
	url := fmt.Sprintf("%s/districts/%s", BaseUrl, ags)
	var response DistrictResponse
	err := httpUtils.GetRequest(
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

func onDistrict(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	_, err := c.EffectiveMessage.Reply(b, districtText(c.Matches[1]), utils.DefaultSendOptions())
	return err
}

func (p *Plugin) setRkiAGS(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	ags := c.Matches[1]

	if len(ags) > 8 {
		_, err := c.EffectiveMessage.Reply(b, "❌ Gemeindeschlüssel muss kleiner als 8 Zeichen lang sein.\nSuche mit <code>/rki STADT</code>.",
			utils.DefaultSendOptions())
		return err
	}

	var response DistrictResponse
	url := fmt.Sprintf("%s/districts/%s", BaseUrl, ags)
	err := httpUtils.GetRequest(
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
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if len(response.Districts) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "❌ Stadt nicht gefunden.", utils.DefaultSendOptions())
		return err
	}

	err = p.rkiService.SetAGS(c.EffectiveUser, ags)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.EffectiveUser.Id).
			Str("ags", ags).
			Msg("error while saving AGS")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, "✅ Du kannst jetzt /myrki nutzen.", utils.DefaultSendOptions())
	return err
}

func (p *Plugin) onMyRKI(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)
	ags, err := p.rkiService.GetAGS(c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.EffectiveUser.Id).
			Msg("error while getting AGS")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if ags == "" {
		_, err := c.EffectiveMessage.Reply(b, "❌ Du hast keinen Gemeindeschlüssel gespeichert."+
			"\nSuche mit <code>/rki STADT</code> und setze ihn mit <code>/setrki_AGS</code>.",
			utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, districtText(ags), utils.DefaultSendOptions())
	return err
}

func (p *Plugin) delRKI(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.rkiService.DelAGS(c.EffectiveUser)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("user_id", c.EffectiveUser.Id).
			Msg("error while deleting AGS")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	_, err = c.EffectiveMessage.Reply(b, "✅", utils.DefaultSendOptions())
	return err
}
