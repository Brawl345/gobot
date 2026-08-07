package tgUtils

import (
	"cmp"
	"errors"
	"os"
	"strconv"
	"sync"

	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var adminId = sync.OnceValue(func() int64 {
	id, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return id
})

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
	return adminId() == user.Id
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
		if photoSize.FileSize > filesize {
			filesize = photoSize.FileSize
			bestResolution = &photoSize
		}
	}

	return bestResolution
}

// EphemeralReplyTo builds the addressing for an answer that only the sender of message and the bot can see.
// The returned receiverUserId is 0 outside of groups, which makes the answer a regular one. An incoming
// ephemeral message carries no message ID, so it is replied to via its ephemeral ID instead.
func EphemeralReplyTo(message *gotgbot.Message) (receiverUserId int64, replyParameters *gotgbot.ReplyParameters) {
	if !FromGroup(message) || message.From == nil {
		return 0, &gotgbot.ReplyParameters{MessageId: message.MessageId, AllowSendingWithoutReply: true}
	}

	if message.EphemeralMessageId != 0 {
		return message.From.Id, &gotgbot.ReplyParameters{EphemeralMessageId: message.EphemeralMessageId}
	}

	return message.From.Id, &gotgbot.ReplyParameters{MessageId: message.MessageId}
}

// ReplyEphemeral replies to a message in a group so that only its sender and the bot can see the answer.
// Outside of groups it falls back to a regular reply.
func ReplyEphemeral(b *gotgbot.Bot, message *gotgbot.Message, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	if opts == nil {
		opts = utils.DefaultSendOptions()
	}

	ephemeralOpts := *opts
	ephemeralOpts.ReceiverUserId, ephemeralOpts.ReplyParameters = EphemeralReplyTo(message)

	return b.SendMessage(message.Chat.Id, text, &ephemeralOpts)
}

type ReactionFallbackOpts struct {
	SendMessageOpts *gotgbot.SendMessageOpts
	Fallback        string
}

// AddReactionWithFallback adds a reaction to a message. If reactions are disabled, a Fallback message is sent instead
func AddReactionWithFallback(b *gotgbot.Bot, message *gotgbot.Message, emoji string, opts *ReactionFallbackOpts) error {
	_, err := message.SetReaction(b, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{
			gotgbot.ReactionTypeEmoji{
				Emoji: emoji,
			},
		},
	})

	if telegramErr, ok := errors.AsType[*gotgbot.TelegramError](err); err != nil && ok && telegramErr.Description == ErrReactionInvalid {
		if opts == nil {
			opts = &ReactionFallbackOpts{}
		}
		fallback := cmp.Or(opts.Fallback, emoji)
		sendMessageOpts := cmp.Or(opts.SendMessageOpts, utils.DefaultSendOptions())

		_, err = message.Reply(b, fallback, sendMessageOpts)
	}

	return err
}
