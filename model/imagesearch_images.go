package model

type ImageSearchImages struct {
	QueryID      int64
	CurrentIndex int
	Images       []ImageSearchImage
}

type ImageSearchImage interface {
	ImageLink() string
	ContextLink() string
	IsGIF() bool
}
