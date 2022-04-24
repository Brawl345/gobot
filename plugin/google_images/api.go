package google_images

import (
	"errors"
	"strings"
)

var (
	ErrNoImagesFound            = errors.New("no images found")
	ErrCouldNotDownloadAnyImage = errors.New("could not download any image")
)

type (
	Response struct {
		Items []GoogleImage `json:"items"`
	}

	GoogleImage struct {
		Link  string `json:"link"`
		Mime  string `json:"mime"`
		Image struct {
			ContextLink string `json:"contextLink"`
		} `json:"image"`
	}
)

func (gi GoogleImage) ImageLink() string {
	return gi.Link
}

func (gi GoogleImage) ContextLink() string {
	return gi.Image.ContextLink
}

func (gi GoogleImage) IsGIF() bool {
	return strings.ToLower(gi.Mime) == "image/gif"
}
