package tgUtils

import (
	"errors"
	"os"
	"strconv"

	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

func AnyEntities(message *gotgbot.Message) []gotgbot.MessageEntity {
	entities := message.Entities
	if message.Entities == nil {
		entities = message.CaptionEntities
	}
	return entities
}

func AnyText(message *gotgbot.Message) string {
	text := message.Text
	if message.Text == "" {
		text = message.Caption
	}
	return text
}

func ParseAnyEntities(message *gotgbot.Message, entity *gotgbot.MessageEntity) gotgbot.ParsedMessageEntity {
	switch {
	case message.Text != "":
		return message.ParseEntity(*entity)
	case message.Caption != "":
		return message.ParseCaptionEntity(*entity)
	default:
		return gotgbot.ParsedMessageEntity{}
	}
}

// ParseAnyEntityTypes is a simplied version of ParseEntityTypes that accepts a slice instead of a map for entites types
// that should be parsed. It also uses caption entites when they exist.
func ParseAnyEntityTypes(message *gotgbot.Message, only []EntityType) []gotgbot.ParsedMessageEntity {
	accepted := make(map[string]struct{}, len(only))
	for _, entityType := range only {
		accepted[string(entityType)] = struct{}{}
	}

	switch {
	case message.Text != "":
		return message.ParseEntityTypes(accepted)
	case message.Caption != "":
		return message.ParseCaptionEntityTypes(accepted)
	default:
		return []gotgbot.ParsedMessageEntity{}
	}
}

func ContainsMedia(message *gotgbot.Message) bool {
	switch {
	case message.Photo != nil:
		return true
	case message.Voice != nil:
		return true
	case message.Audio != nil:
		return true
	case message.Animation != nil:
		return true
	case message.Sticker != nil:
		return true
	case message.Document != nil:
		return true
	case message.Video != nil:
		return true
	case message.VideoNote != nil:
		return true
	default:
		return false
	}
}

func IsAdmin(user *gotgbot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.Id
}

func FromGroup(message gotgbot.MaybeInaccessibleMessage) bool {
	return message.GetChat().Type == gotgbot.ChatTypeGroup || message.GetChat().Type == gotgbot.ChatTypeSupergroup
}

func IsPrivate(message *gotgbot.Message) bool {
	return message.Chat.Type == gotgbot.ChatTypePrivate
}

func IsReply(message *gotgbot.Message) bool {
	return message.ReplyToMessage != nil
}

func GetBestResolution(photo []gotgbot.PhotoSize) *gotgbot.PhotoSize {
	if photo == nil {
		return nil
	}
	var filesize int64
	var bestResolution *gotgbot.PhotoSize
	for _, photoSize := range photo {
		photoSize := photoSize
		if photoSize.FileSize > filesize {
			filesize = photoSize.FileSize
			bestResolution = &photoSize
		}
	}

	return bestResolution
}

type ReactionFallbackOpts struct {
	SendMessageOpts *gotgbot.SendMessageOpts
	Fallback        string
}

// AddRectionWithFallback adds a reaction to a message. If reactions are disabled, a Fallback message is sent instead
func AddRectionWithFallback(b *gotgbot.Bot, message *gotgbot.Message, emoji string, opts *ReactionFallbackOpts) error {
	_, err := message.SetReaction(b, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{
			gotgbot.ReactionTypeEmoji{
				Emoji: emoji,
			},
		},
	})

	var telegramErr *gotgbot.TelegramError
	if err != nil && errors.As(err, &telegramErr) && telegramErr.Description == ErrReactionInvalid {
		fallback := opts.Fallback
		if fallback == "" {
			fallback = emoji
		}

		sendMessageOpts := opts.SendMessageOpts
		if sendMessageOpts == nil {
			sendMessageOpts = utils.DefaultSendOptions()
		}

		_, err = message.Reply(b, fallback, sendMessageOpts)
	}

	return err
}
