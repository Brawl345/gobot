package anilist

import (
	"fmt"
	"github.com/Brawl345/gobot/utils"
	"html"
	"regexp"
	"strings"
	"time"
)

const (
	RankThreshold = 60
)

var descriptionRegex = regexp.MustCompile(`<.*?>`)

func (anime *MediaByIdMedia) DescriptionCleaned() string {
	description := html.UnescapeString(anime.Description)
	description = descriptionRegex.ReplaceAllString(description, "")
	return description
}

// RelevantTags returns all tags that are not spoilers and ranked with more than RankThreshold points
func (anime *MediaByIdMedia) RelevantTags() []MediaByIdMediaTagsMediaTag {
	var tags []MediaByIdMediaTagsMediaTag

	for _, tag := range anime.Tags {
		if !tag.IsGeneralSpoiler && tag.Rank >= RankThreshold {
			tags = append(tags, tag)
		}
	}

	return tags
}

func (anime *MediaByIdMedia) SeasonFormatted() string {
	switch anime.Season {
	case MediaSeasonSpring:
		return "Frühling"
	case MediaSeasonSummer:
		return "Sommer"
	case MediaSeasonFall:
		return "Herbst"
	case MediaSeasonWinter:
		return "Winter"
	default:
		return string(anime.Season)
	}
}

func (anime *MediaByIdMedia) StatusFormatted() string {
	switch anime.Status {
	case MediaStatusFinished:
		return "Beendet"
	case MediaStatusReleasing:
		return "Läuft zurzeit"
	case MediaStatusNotYetReleased:
		return "In Zukunft"
	case MediaStatusCancelled:
		return "Abgebrochen"
	case MediaStatusHiatus:
		return "Unterbrochen"
	default:
		return string(anime.Status)
	}
}

func (startDate *MediaByIdMediaStartDateFuzzyDate) Formatted() string {
	if startDate.Day == 0 && startDate.Month == 0 && startDate.Year == 0 { // // Not yet aired and no date known
		return ""
	}

	day := startDate.Day
	if day == 0 {
		day = 1
	}

	date := time.Date(startDate.Year, time.Month(startDate.Month), day, 0, 0, 0, 0, time.UTC)

	if startDate.Day != 0 { // All
		return date.Format("02.01.2006")
	} else if startDate.Month != 0 { // Only month and year
		return utils.LocalizeDatestring(date.Format("January 2006"))
	}

	return date.Format("2006")
}
func (endDate *MediaByIdMediaEndDateFuzzyDate) Formatted() string {
	if endDate.Year != 0 && endDate.Month != 0 && endDate.Day != 0 {
		return fmt.Sprintf("%02d.%02d.%d", endDate.Day, endDate.Month, endDate.Year)
	}

	return ""
}

func (format MediaFormat) String() string {
	switch format {
	case MediaFormatTv:
		return "TV"
	case MediaFormatMovie:
		return "Film"
	case MediaFormatSpecial:
		return "Special"
	case MediaFormatOva:
		return "OVA"
	case MediaFormatOna:
		return "ONA"
	case MediaFormatMusic:
		return "Musik"
	default:
		return string(format)
	}
}

func (titles *MediaByIdMediaTitle) AlternativeTitles() []string {
	var alternativeTitles []string

	if titles.Native != "" && strings.ToLower(titles.Native) != strings.ToLower(titles.Romaji) {
		alternativeTitles = append(alternativeTitles, titles.Native)
	}

	if titles.English != "" && strings.ToLower(titles.English) != strings.ToLower(titles.Romaji) {
		alternativeTitles = append(alternativeTitles, titles.English)
	}

	return alternativeTitles
}
