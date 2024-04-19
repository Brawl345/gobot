package tgUtils

// EntityType is one of https://core.telegram.org/bots/api#messageentity
type EntityType string

type MessageTrigger string

const (
	AnyMedia    MessageTrigger = "\agobot_media"
	AnyMsg      MessageTrigger = "\agobot_msg"
	DocumentMsg MessageTrigger = "\agobot_document"
	LocationMsg MessageTrigger = "\agobot_location"
	PhotoMsg    MessageTrigger = "\agobot_photo"
	VenueMsg    MessageTrigger = "\agobot_venue"
	VoiceMsg    MessageTrigger = "\agobot_voice"

	MaxCaptionLength            = 1024
	MaxMessageLength            = 4096
	MaxFilesizeDownload         = 20000000 // Max filesize that can be downloaded from Telegram = 20MB
	MaxFilesizeUpload           = 50000000 // Max filesize that can be uploaded to Telegram = 50MB
	MaxPhotosizeUpload          = 10000000 // Max filesize of photos that can be uploaded to Telegram = 10 MB
	MaxPhotosizeThroughTelegram = 5000000  // Max filesize of photos that Telegram can send automatically = 5 MB

	ChatActionFindLocation        = "find_location"
	ChatActionUploadDocument      = "upload_document"
	ChatActionUploadPhoto         = "upload_photo"
	ChatActionUploadVideo         = "upload_video"
	ChatActionTyping              = "typing"
	ChatMemberStatusCreator       = "creator"
	ChatMemberStatusAdministrator = "administrator"

	EntityTextLink    EntityType = "text_link"
	EntityTypeMention EntityType = "mention"
	EntityTypeURL     EntityType = "url"

	ErrBlockedByUser     = "Forbidden: bot was blocked by the user"
	ErrReactionInvalid   = "Bad Request: REACTION_INVALID"
	ErrNotStartedByUser  = "Forbidden: bot can't initiate conversation with a user"
	ErrUserIsDeactivated = "Forbidden: user is deactivated"
)
