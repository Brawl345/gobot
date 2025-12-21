package gelbooru

import (
	"fmt"
	"strings"
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
	if p.Source != "" {
		sb.WriteString(fmt.Sprintf(" - 🌐 <a href=\"%s\">Quelle</a>\n", p.Source))
	}

	return sb.String()
}

// AltCaption is used when the media is too big or invalid type
func (p *Post) AltCaption() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", p.DirectURL()))
	if p.IsNSFW() {
		sb.WriteString("️🔞 <b>NSFW</b> - ")
	}
	sb.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Post #%d</a>", p.PostURL(), p.Id))
	if p.Source != "" {
		sb.WriteString(fmt.Sprintf(" - 🌐 <a href=\"%s\">Quelle</a>\n", p.Source))
	}

	return sb.String()
}

func (p *Post) PostURL() string {
	return fmt.Sprintf(PostURL, p.Id)
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
	if strings.HasSuffix(p.FileURL(), ".gif") {
		return true
	}
	return false
}
