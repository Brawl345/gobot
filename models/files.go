package models

type FileService interface {
	Create(uniqueID, fileName, mediaType string) error
	Exists(uniqueID string) (bool, error)
}
