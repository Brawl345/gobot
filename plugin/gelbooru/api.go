package gelbooru

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

const (
	PostURL = "https://gelbooru.com/index.php?page=post&s=view&id=%d"
)

type (
	Post struct {
		Id            int    `json:"id"`
		CreatedAt     string `json:"created_at"`
		Score         int    `json:"score"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
		Md5           string `json:"md5"`
		Directory     string `json:"directory"`
		Image         string `json:"image"`
		Rating        string `json:"rating"`
		Source        string `json:"source"`
		Change        int    `json:"change"`
		Owner         string `json:"owner"`
		CreatorId     int    `json:"creator_id"`
		ParentId      int    `json:"parent_id"`
		Sample        int    `json:"sample"`
		PreviewHeight int    `json:"preview_height"`
		PreviewWidth  int    `json:"preview_width"`
		Tags          string `json:"tags"`
		Title         string `json:"title"`
		HasNotes      string `json:"has_notes"`
		HasComments   string `json:"has_comments"`
		FileUrl       string `json:"file_url"`
		PreviewUrl    string `json:"preview_url"`
		SampleUrl     string `json:"sample_url"`
		SampleHeight  int    `json:"sample_height"`
		SampleWidth   int    `json:"sample_width"`
		Status        string `json:"status"`
		PostLocked    int    `json:"post_locked"`
		HasChildren   string `json:"has_children"`
	}

	Response struct {
		Attributes struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
			Count  int `json:"count"`
		} `json:"@attributes"`
		Post []Post `json:"post"`
	}
)

func (p *Post) FileURL() string {
	// For GIFs, the sample URL is a static JPG which we don't want
	if p.IsGIF() {
		return p.FileUrl
	}

	if p.SampleUrl != "" {
		return p.SampleUrl
	}
	return p.FileUrl
}

func (p *Post) DirectURL() string {
	if p.FileUrl != "" {
		return p.FileUrl
	}
	return p.SampleUrl
}

func (p *Post) IsNSFW() bool {
	if p.Rating == "questionable" || p.Rating == "explicit" {
		return true
	}
	return false
}

func (p *Post) Caption() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Post #%d</a> - ", p.PostURL(), p.Id))
	sb.WriteString(fmt.Sprintf("🖼️ <a href=\"%s\">Direktlink</a>", p.DirectURL()))
	sb.WriteString(sourceLinks(p.ValidSources()))

	return sb.String()
}

// sourceLinks renders one or more source links using the host as link text.
func sourceLinks(sources []string) string {
	if len(sources) == 0 {
		return ""
	}

	label := "Quelle"
	if len(sources) > 1 {
		label = "Quellen"
	}

	links := make([]string, len(sources))
	for i, src := range sources {
		host := src
		if parsedURL, err := url.Parse(src); err == nil {
			host = strings.TrimPrefix(parsedURL.Host, "www.")
		}
		links[i] = fmt.Sprintf("<a href=\"%s\">%s</a>", src, host)
	}

	return fmt.Sprintf(" - 🌐 %s: %s\n", label, strings.Join(links, ", "))
}

// AltCaption is used when the media is too big or invalid type
func (p *Post) AltCaption() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", p.DirectURL()))
	if p.IsNSFW() {
		sb.WriteString("️🔞 <b>NSFW</b> - ")
	}
	sb.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Post #%d</a>", p.PostURL(), p.Id))
	sb.WriteString(sourceLinks(p.ValidSources()))

	return sb.String()
}

func (p *Post) PostURL() string {
	return fmt.Sprintf(PostURL, p.Id)
}

// ValidSources returns all valid HTTP(S) source URLs. Gelbooru separates
// multiple sources by whitespace or "|".
func (p *Post) ValidSources() []string {
	if p.Source == "" {
		return nil
	}

	parts := strings.FieldsFunc(p.Source, func(r rune) bool {
		return unicode.IsSpace(r) || r == '|'
	})

	var sources []string
	for _, src := range parts {
		parsedURL, err := url.Parse(src)
		if err != nil {
			continue
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			continue
		}

		if parsedURL.Host == "" {
			continue
		}

		sources = append(sources, src)
	}

	return sources
}

func (p *Post) IsImage() bool {
	if strings.HasSuffix(p.FileURL(), ".jpg") ||
		strings.HasSuffix(p.FileURL(), ".jpeg") ||
		strings.HasSuffix(p.FileURL(), ".png") ||
		strings.HasSuffix(p.FileURL(), ".webp") {
		return true
	}
	return false
}

func (p *Post) IsVideo() bool {
	if strings.HasSuffix(p.FileURL(), ".mp4") ||
		strings.HasSuffix(p.FileURL(), ".webm") {
		return true
	}
	return false
}

func (p *Post) IsGIF() bool {
	if strings.HasSuffix(p.SampleUrl, ".gif") || strings.HasSuffix(p.FileUrl, ".gif") {
		return true
	}
	return false
}
