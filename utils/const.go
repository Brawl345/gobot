package utils

import "time"

// EntityType is one of https://core.telegram.org/bots/api#messageentity
type EntityType string

type MessageTrigger string

const (
	AnyMsg      MessageTrigger = "\agobot_msg"
	DocumentMsg MessageTrigger = "\agobot_document"
	LocationMsg MessageTrigger = "\agobot_location"
	PhotoMsg    MessageTrigger = "\agobot_photo"
	VenueMsg    MessageTrigger = "\agobot_venue"
	AnyMedia    MessageTrigger = "\agobot_media"

	Day  = 24 * time.Hour
	Week = 7 * Day

	InlineQueryFailureCacheTime = 2 // In seconds

	MaxMessageLength            = 4096
	MaxFilesizeDownload         = 20000000 // Max filesize that can be downloaded from Telegram = 20MB
	MaxFilesizeUpload           = 50000000 // Max filesize that can be uploaded to Telegram = 50MB
	MaxPhotosizeUpload          = 10000000 // Max filesize of photos that can be uploaded to Telegram = 10 MB
	MaxPhotosizeThroughTelegram = 5000000  // Max filesize of photos that Telegram can send automatically = 5 MB

	ChatActionFindLocation   = "find_location"
	ChatActionUploadDocument = "upload_document"
	ChatActionUploadPhoto    = "upload_photo"
	ChatActionTyping         = "typing"

	ChatMemberStatusCreator       = "creator"
	ChatMemberStatusAdministrator = "administrator"

	EntityTextLink    EntityType = "text_link"
	EntityTypeMention EntityType = "mention"
	EntityTypeURL     EntityType = "url"

	ErrBlockedByUser     = "Forbidden: bot was blocked by the user"
	ErrNotStartedByUser  = "Forbidden: bot can't initiate conversation with a user"
	ErrUserIsDeactivated = "Forbidden: user is deactivated"

	UserAgent = "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
)
