package model

type GoogleImages struct {
	QueryID      int64
	CurrentIndex int
	Images       []Image
}

type Image interface {
	ImageLink() string
	ContextLink() string
	IsGIF() bool
}
