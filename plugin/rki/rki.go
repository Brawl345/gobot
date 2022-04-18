package rki

import (
	"fmt"
	"html"
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

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "rki"
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
				html.EscapeString(district.Name),
				html.EscapeString(district.County),
				html.EscapeString(district.State),
			),
		)
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func onDistrict(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	var response DistrictResponse

	url := fmt.Sprintf("%s/districts/%s", BaseUrl, c.Matches[1])
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

	district := response.Districts[c.Matches[1]]

	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>COVID-19-Übersicht für %s (%s, %s) lt. RKI:</b>\n",
			html.EscapeString(district.Name),
			html.EscapeString(district.County),
			html.EscapeString(district.State),
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
			"\n<i>Zuletzt aktualisiert: %s</i>",
			response.Meta.LastUpdate.In(timezone).Format("02.01.2006"),
		),
	)

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}
