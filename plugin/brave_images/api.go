package brave_images

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
		Results []BraveImage `json:"results"`
	}

	BraveImage struct {
		Url        string `json:"url"`
		Properties struct {
			Url string `json:"url"`
		} `json:"properties"`
		Confidence string `json:"confidence"`
	}
)

func (bi BraveImage) ImageLink() string {
	return bi.Properties.Url
}

func (bi BraveImage) ContextLink() string {
	return bi.Url
}

func (bi BraveImage) IsGIF() bool {
	// No other way to check this sadly
	return strings.HasSuffix(strings.ToLower(bi.Properties.Url), ".gif")
}
