package anilist

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	RankThreshold = 60
)

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

func getMonthName(month int) string {
	months := []string{
		"Januar", "Februar", "März",
		"April", "Mai", "Juni",
		"Juli", "August", "September",
		"Oktober", "November", "Dezember",
	}

	if month < 1 || month > 12 {
		return "Da ist ein Bug in deinem Code"
	}

	return months[month-1]
}

func (startDate *MediaByIdMediaStartDateFuzzyDate) Formatted() string {
	if startDate.Day == 0 && startDate.Month == 0 && startDate.Year == 0 { // // Not yet aired and no date known
		return ""
	}

	if startDate.Year != 0 && startDate.Month != 0 && startDate.Day != 0 { // All
		return fmt.Sprintf("%02d.%02d.%d", startDate.Day, startDate.Month, startDate.Year)
	} else if startDate.Day == 0 && startDate.Month == 0 { // Only year
		return strconv.Itoa(startDate.Year)
	} else if startDate.Day == 0 { // Only month and year
		return fmt.Sprintf("%s %d", getMonthName(startDate.Month), startDate.Year)
	}

	return ""
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
