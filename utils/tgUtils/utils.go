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

func ContainsMedia(m *gotgbot.Message) bool {
	switch {
	case m.Photo != nil:
		return true
	case m.Voice != nil:
		return true
	case m.Audio != nil:
		return true
	case m.Animation != nil:
		return true
	case m.Sticker != nil:
		return true
	case m.Document != nil:
		return true
	case m.Video != nil:
		return true
	case m.VideoNote != nil:
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
