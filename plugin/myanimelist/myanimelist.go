package myanimelist

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("myanimelist")

const (
	SynopsisThreshold = 250
)

type Plugin struct {
	clientID string
}

func New(credentialService model.CredentialService) *Plugin {
	clientID, err := credentialService.GetKey("mal_client_id")
	if err != nil {
		log.Warn().Msg("mal_client_id not found")
	}

	return &Plugin{
		clientID: clientID,
	}
}

func (*Plugin) Name() string {
	return "myanimelist"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "mal",
			Description: "<Suchbegriff> - Anime suchen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/mal(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.onSearch,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/mal_(\d+)(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onAnime,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`myanimelist\.net/anime/(\d+)`),
			HandlerFunc: p.onAnime,
		},
	}
}

func (p *Plugin) onSearch(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	var response AnimeSearch

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "api.myanimelist.net",
		Path:   "/v2/anime",
	}
	q := requestUrl.Query()
	q.Set("q", c.Matches[1])
	q.Set("fields", "id,title,nsfw,rating")
	q.Set("limit", "5")
	q.Set("nsfw", "true")
	requestUrl.RawQuery = q.Encode()

	err := httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"X-MAL-CLIENT-ID": p.clientID,
		},
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", requestUrl.String()).
			Msg("error getting myanimelist search results")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.Results) == 0 {
		return c.Reply("❌ Es wurde kein Anime gefunden.", utils.DefaultSendOptions)
	}

	var sb strings.Builder

	for _, result := range response.Results {
		sb.WriteString(
			fmt.Sprintf(
				"/mal_%d - <a href=\"https://myanimelist.net/anime/%d/\"><b>%s</b></a>",
				result.Anime.ID,
				result.Anime.ID,
				utils.Escape(result.Anime.Title),
			),
		)
		if result.Anime.NSFW() {
			sb.WriteString(" <i>(NSFW)</i>")
		}
		sb.WriteString("\n")
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func (p *Plugin) onAnime(c plugin.GobotContext) error {
	_ = c.Notify(telebot.Typing)
	var anime Anime
	var httpError *httpUtils.HttpError

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "api.myanimelist.net",
		Path:   fmt.Sprintf("/v2/anime/%s", c.Matches[1]),
	}
	q := requestUrl.Query()
	q.Set("fields", "id,title,main_picture,alternative_titles,start_date,end_date,synopsis,mean,rank,popularity,nsfw,media_type,status,genres,num_episodes,start_season,average_episode_duration,rating,studios")
	requestUrl.RawQuery = q.Encode()

	err := httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"X-MAL-CLIENT-ID": p.clientID,
		},
		&anime,
	)

	if err != nil {
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 404 {
				return c.Reply("❌ Anime nicht gefunden.", utils.DefaultSendOptions)
			}
		}

		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", requestUrl.String()).
			Msg("error getting myanimelist result")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	var sb strings.Builder
	disableWebPagePreview := true

	// Main Picture
	if anime.GetMainPicture() != "" && !anime.NSFW() {
		disableWebPagePreview = false
		sb.WriteString(utils.EmbedImage(anime.GetMainPicture()))
	}

	// Title
	sb.WriteString(
		fmt.Sprintf(
			"<a href=\"https://myanimelist.net/anime/%d/\"><b>%s</b></a>",
			anime.ID,
			utils.Escape(anime.Title),
		),
	)

	// Alternative Titles
	alternativeTitles := anime.GetAlternativeTitles()
	if len(alternativeTitles) > 0 {
		sb.WriteString(" (<i>")
		sb.WriteString(utils.Escape(strings.Join(alternativeTitles, ", ")))
		sb.WriteString("</i>)")
	}

	// Type
	sb.WriteString(
		fmt.Sprintf(
			" [%s]",
			utils.Escape(anime.GetMediaType()),
		),
	)

	// Not safe for work
	if anime.NSFW() {
		sb.WriteString("\n😱 <strong>NSFW</strong>")
	}

	sb.WriteString("\n")

	// Studios
	if len(anime.Studios) > 0 {
		plural := ""
		if len(anime.Studios) > 1 {
			plural = "s"
		}
		sb.WriteString(
			fmt.Sprintf(
				"🎨 <b>Studio%s:</b> ",
				plural,
			),
		)

		for i, studio := range anime.Studios {
			sb.WriteString(
				fmt.Sprintf(
					"<a href=\"https://myanimelist.net/anime/producer/%d/\">%s</a>",
					studio.ID,
					utils.Escape(studio.Name),
				),
			)
			if i < len(anime.Studios)-1 {
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
					"<a href=\"https://myanimelist.net/anime/genre/%d/\">%s</a>",
					genre.ID,
					utils.Escape(genre.Name),
				),
			)
			if i < len(anime.Genres)-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString("\n")
	}

	// Episodes
	if anime.NumEpisodes > 0 {
		sb.WriteString(
			fmt.Sprintf(
				"📺 <b>Episoden:</b> %d",
				anime.NumEpisodes,
			),
		)

		if anime.AverageEpisodeDuration >= 60 {
			sb.WriteString(
				fmt.Sprintf(
					" <i>(%d Minuten pro Episode)</i>",
					anime.AverageEpisodeDuration/60,
				),
			)
		}

		sb.WriteString("\n")
	}

	// Airing
	var hasAiredInfo bool
	if anime.StartDate != "" {
		startDate, err := anime.StartDateFormatted()
		if err != nil {
			log.Error().
				Err(err).
				Str("startDate", anime.StartDate).
				Str("url", requestUrl.String()).
				Msg("error parsing startDate")
		} else {
			hasAiredInfo = true
			sb.WriteString(
				fmt.Sprintf(
					"📆 <b>Ausstrahlung:</b> %s",
					startDate,
				),
			)

			if anime.EndDate != "" && anime.StartDate != anime.EndDate {
				endDate, err := anime.EndDateFormatted()
				if err != nil {
					log.Error().
						Err(err).
						Str("endDate", anime.EndDate).
						Str("url", requestUrl.String()).
						Msg("error parsing endDate")
				} else {
					sb.WriteString(
						fmt.Sprintf(
							" bis %s",
							endDate,
						),
					)
				}
			}
		}

	} else if anime.StartSeason.Year > 0 {
		hasAiredInfo = true
		sb.WriteString(
			fmt.Sprintf(
				"📆 <b>Ausstrahlung:</b> %s %d",
				utils.Escape(anime.StartSeason.Season),
				anime.StartSeason.Year,
			),
		)
	}

	// Status
	if hasAiredInfo {
		sb.WriteString(
			fmt.Sprintf(
				" <i>(%s)</i>\n",
				utils.Escape(anime.GetStatus()),
			),
		)
	} else {
		sb.WriteString(
			fmt.Sprintf(
				"📆 <b>Ausstrahlung:</b> <i>%s</i>\n",
				utils.Escape(anime.GetStatus()),
			),
		)
	}

	// Rating
	if anime.Mean > 0 {
		mean := fmt.Sprintf("%.2f", anime.Mean)
		meanString := strings.ReplaceAll(mean, ".", ",")
		sb.WriteString(
			fmt.Sprintf(
				"⭐ <b>Bewertung:</b> %s ",
				meanString,
			),
		)

		if anime.Rank > 0 && anime.Popularity > 0 {
			sb.WriteString(
				fmt.Sprintf(
					"<i>(Platz #%s, Popularität #%s)</i>",
					utils.FormatThousand(anime.Rank),
					utils.FormatThousand(anime.Popularity),
				),
			)
		}

		sb.WriteString("\n")
	}

	// Synopsis
	if anime.Synopsis != "" {
		sb.WriteString("\n")
		if len(anime.Synopsis) > SynopsisThreshold {
			sb.WriteString(utils.Escape(anime.Synopsis[:SynopsisThreshold]))
			sb.WriteString("...")
		} else {
			sb.WriteString(utils.Escape(anime.Synopsis))
		}
	}

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: disableWebPagePreview,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
	})

}
