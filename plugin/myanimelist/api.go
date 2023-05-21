package myanimelist

import (
	"strings"
	"time"

	"github.com/Brawl345/gobot/utils"
)

type (
	Anime struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		MainPicture struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
		} `json:"main_picture"`
		AlternativeTitles struct {
			Synonyms []string `json:"synonyms"`
			En       string   `json:"en"`
			Ja       string   `json:"ja"`
		} `json:"alternative_titles"`
		StartDate  string  `json:"start_date"`
		EndDate    string  `json:"end_date"`
		Synopsis   string  `json:"synopsis"`
		Mean       float64 `json:"mean"`
		Rank       int     `json:"rank"`
		Popularity int     `json:"popularity"`
		Nsfw       string  `json:"nsfw"`
		MediaType  string  `json:"media_type"`
		Status     string  `json:"status"`
		Genres     []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
		NumEpisodes int `json:"num_episodes"`
		StartSeason struct {
			Year   int    `json:"year"`
			Season string `json:"season"`
		} `json:"start_season"`
		AverageEpisodeDuration int `json:"average_episode_duration"`
		Studios                []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"studios"`
	}

	AnimeSearch struct {
		Results []struct {
			Anime AnimeResult `json:"node"`
		} `json:"data"`
	}

	// AnimeResult is an extra struct because we don't need all the fields
	AnimeResult struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Nsfw  string `json:"nsfw"`
	}
)

func (a *AnimeResult) NSFW() bool {
	return a.Nsfw == "gray" || a.Nsfw == "black"
}

func (a *Anime) GetMainPicture() string {
	if a.MainPicture.Large != "" {
		return a.MainPicture.Large
	}
	return a.MainPicture.Medium
}

func (a *Anime) NSFW() bool {
	return a.Nsfw == "gray" || a.Nsfw == "black"
}

func (a *Anime) GetAlternativeTitles() []string {
	var titles []string
	if a.AlternativeTitles.En != "" && a.AlternativeTitles.En != a.Title {
		titles = append(titles, a.AlternativeTitles.En)
	}
	if a.AlternativeTitles.Ja != "" {
		titles = append(titles, a.AlternativeTitles.Ja)
	}
	titles = append(titles, a.AlternativeTitles.Synonyms...)
	return titles
}

func (a *Anime) GetMediaType() string {
	switch a.MediaType {
	case "tv":
		return "TV"
	case "ova":
		return "OVA"
	case "ona":
		return "ONA"
	case "movie":
		return "Film"
	case "special":
		return "Special"
	case "music":
		return "Musik"
	case "unknown":
		return "Unbekannt"
	default:
		return a.MediaType
	}
}

func (a *Anime) GetStatus() string {
	switch a.Status {
	case "finished_airing":
		return "Beendet"
	case "currently_airing":
		return "Läuft zurzeit"
	case "not_yet_aired":
		return "In Zukunft"
	default:
		return a.Status
	}
}

func (a *Anime) GetSeason() string {
	switch a.StartSeason.Season {
	case "spring":
		return "Frühling"
	case "summer":
		return "Sommer"
	case "fall":
		return "Herbst"
	case "winter":
		return "Winter"
	default:
		return a.StartSeason.Season
	}
}

func (a *Anime) StartDateFormatted() (string, error) {
	if a.StartDate == "" {
		return "", nil
	}
	if strings.Count(a.StartDate, "-") == 2 {
		parsed, err := time.Parse("2006-01-02", a.StartDate)
		if err != nil {
			return "", err
		}
		return parsed.Format("02.01.2006"), nil
	} else if strings.Count(a.StartDate, "-") == 1 {
		parsed, err := time.Parse("2006-01", a.StartDate)
		if err != nil {
			return "", err
		}
		return utils.LocalizeDatestring(parsed.Format("January 2006")), nil
	}
	parsed, err := time.Parse("2006", a.StartDate)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006"), nil
}

func (a *Anime) EndDateFormatted() (string, error) {
	if a.EndDate == "" {
		return "", nil
	}
	if strings.Count(a.EndDate, "-") == 2 {
		parsed, err := time.Parse("2006-01-02", a.EndDate)
		if err != nil {
			return "", err
		}
		return parsed.Format("02.01.2006"), nil
	} else if strings.Count(a.EndDate, "-") == 1 {
		parsed, err := time.Parse("2006-01", a.EndDate)
		if err != nil {
			return "", err
		}
		return utils.LocalizeDatestring(parsed.Format("January 2006")), nil
	}
	parsed, err := time.Parse("2006", a.EndDate)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006"), nil
}
