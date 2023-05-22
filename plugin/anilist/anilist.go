package anilist

import (
	"context"
	"fmt"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Khan/genqlient/graphql"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var log = logger.New("anilist")

const (
	ApiUrl               = "https://graphql.anilist.co"
	MaxTags              = 7
	DescriptionThreshold = 350
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "anilist"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "al",
			Description: "<Suchbegriff> - Anime auf AniList suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/al(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onSearch,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/al_(\d+)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onAnime,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`anilist\.co/anime/(\d+)`),
			HandlerFunc: p.onAnime,
		},
	}
}

func (p *Plugin) onSearch(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	query := c.Matches[1]

	ctx := context.Background()
	client := graphql.NewClient(ApiUrl, http.DefaultClient)
	resp, err := SearchMediaByTitle(ctx, client, query)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("query", query).
			Msg("error while contacting AniList GraphQL server")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	results := resp.Page.Media

	if len(results) == 0 {
		return c.Reply("❌ Es wurde kein Anime gefunden.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	for _, result := range results {
		sb.WriteString(
			fmt.Sprintf(
				"/al_%d - <a href=\"%s\"><b>%s</b></a>",
				result.Id,
				result.SiteUrl,
				utils.Escape(result.Title.Romaji),
			),
		)
		if result.IsAdult {
			sb.WriteString(" <i>(NSFW)</i>")
		}
		sb.WriteString("\n")
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func (p *Plugin) onAnime(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)

	animeId, err := strconv.ParseInt(c.Matches[1], 10, 32)
	if err != nil {
		log.Warn().
			Err(err).
			Str("animeId", c.Matches[1]).
			Msg("error casting string to int")
		return nil
	}

	ctx := context.Background()
	client := graphql.NewClient(ApiUrl, http.DefaultClient)
	resp, err := MediaById(ctx, client, int(animeId))

	if err != nil {
		// This: https://github.com/Khan/genqlient/blob/main/docs/FAQ.md#-handle-graphql-errors does not work
		// So we just parse the string...
		if strings.HasPrefix(err.Error(), "returned error 404") {
			return c.Reply("❌ Anime nicht gefunden.", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("animeId", c.Matches[1]).
			Msg("error while contacting AniList GraphQL server")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var sb strings.Builder
	disableWebPagePreview := true
	anime := resp.Media

	// Main Picture
	if anime.CoverImage.ExtraLarge != "" && !anime.IsAdult {
		disableWebPagePreview = false
		sb.WriteString(utils.EmbedImage(anime.CoverImage.ExtraLarge))
	}

	// Title
	sb.WriteString(
		fmt.Sprintf(
			"<a href=\"%s\"><b>%s</b></a>",
			anime.SiteUrl,
			utils.Escape(anime.Title.Romaji),
		),
	)

	// Alternative Titles
	alternativeTitles := anime.Title.AlternativeTitles()
	if len(alternativeTitles) > 0 {
		sb.WriteString(" (<i>")
		sb.WriteString(utils.Escape(strings.Join(alternativeTitles, ", ")))
		sb.WriteString("</i>)")
	}

	// Type/Format
	format := anime.Format.String()
	if format != "" {
		sb.WriteString(
			fmt.Sprintf(
				" [%s]",
				utils.Escape(anime.Format.String()),
			),
		)
	}

	// Not safe for work
	if anime.IsAdult {
		sb.WriteString("\n😱 <strong>NSFW</strong>")
	}

	sb.WriteString("\n")

	// Studios
	if len(anime.Studios.Nodes) > 0 {
		plural := ""
		if len(anime.Studios.Nodes) > 1 {
			plural = "s"
		}
		sb.WriteString(
			fmt.Sprintf(
				"🎨 <b>Studio%s:</b> ",
				plural,
			),
		)

		for i, studio := range anime.Studios.Nodes {
			sb.WriteString(
				fmt.Sprintf(
					"<a href=\"https://anilist.co/studio/%d/\">%s</a>",
					studio.Id,
					utils.Escape(studio.Name),
				),
			)
			if i < len(anime.Studios.Nodes)-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString("\n")
	}

	// Genres
	if len(anime.Genres) > 0 {
		plural := ""
		if len(anime.Genres) > 1 {
			plural = "s"
		}
		sb.WriteString(
			fmt.Sprintf(
				"📚 <b>Genre%s:</b> ",
				plural,
			),
		)

		for i, genre := range anime.Genres {
			sb.WriteString(
				fmt.Sprintf(
					"<a href=\"https://anilist.co/search/anime/%s\">%s</a>",
					url.PathEscape(genre),
					utils.Escape(genre),
				),
			)
			if i < len(anime.Genres)-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString("\n")
	}

	// Tags
	tags := anime.RelevantTags()
	if len(tags) > 0 {
		plural := ""
		if len(tags) > 1 {
			plural = "s"
		}
		sb.WriteString(
			fmt.Sprintf(
				"🏷 <b>Tag%s:</b> ",
				plural,
			),
		)

		for i, tag := range tags {
			if i == MaxTags {
				sb.WriteString("...")
				break
			}
			sb.WriteString(
				fmt.Sprintf(
					"<a href=\"https://anilist.co/search/anime?genres=%s\">%s</a>",
					url.QueryEscape(tag.Name),
					utils.Escape(tag.Name),
				),
			)
			if i < len(tags)-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString("\n")
	}

	// Episodes
	if anime.Episodes > 0 {
		sb.WriteString(
			fmt.Sprintf(
				"📺 <b>Episoden:</b> %d",
				anime.Episodes,
			),
		)

		sb.WriteString(
			fmt.Sprintf(
				" <i>(%d Minuten pro Episode)</i>",
				anime.Duration,
			),
		)

		sb.WriteString("\n")
	}

	// Airing
	var hasAiredInfo bool
	startDate := anime.StartDate.Formatted()
	if startDate != "" {
		hasAiredInfo = true

		// Special case when only the year is given => see if we can show the airing season
		if anime.StartDate.Day == 0 && anime.StartDate.Month == 0 && anime.StartDate.Year != 0 && anime.Season != "" {
			startDate = fmt.Sprintf("%s %d", anime.SeasonFormatted(), anime.SeasonYear)
		}

		sb.WriteString(
			fmt.Sprintf(
				"📆 <b>Ausstrahlung:</b> %s",
				startDate,
			),
		)

		endDate := anime.EndDate.Formatted()
		if endDate != "" && startDate != endDate {
			sb.WriteString(
				fmt.Sprintf(
					" bis %s",
					endDate,
				),
			)
		}
	}

	// Status
	if hasAiredInfo {
		sb.WriteString(
			fmt.Sprintf(
				" <i>(%s)</i>\n",
				utils.Escape(anime.StatusFormatted()),
			),
		)
	} else {
		sb.WriteString(
			fmt.Sprintf(
				"📆 <b>Ausstrahlung:</b> <i>%s</i>\n",
				utils.Escape(anime.StatusFormatted()),
			),
		)
	}

	// Rating
	if anime.AverageScore > 0 {
		sb.WriteString(
			fmt.Sprintf(
				"⭐ <b>Bewertung:</b> %d ",
				anime.AverageScore,
			),
		)

		// TODO: rankings (MediaRank)
		if anime.Popularity > 0 {
			sb.WriteString(
				fmt.Sprintf(
					"<i>(Auf %s Listen)</i>",
					utils.FormatThousand(anime.Popularity),
				),
			)
		}

		sb.WriteString("\n")
	}

	// Synopsis/Description
	if anime.Description != "" {
		sb.WriteString("\n")
		description := anime.DescriptionCleaned()
		if len(description) > DescriptionThreshold {
			sb.WriteString(utils.Escape(description[:DescriptionThreshold]))
			sb.WriteString("...")
		} else {
			sb.WriteString(utils.Escape(description))
		}
	}

	// TODO: More fields?
	// TODO: Cleanup unused GraphQL scheme fields

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: disableWebPagePreview,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
	})
}
