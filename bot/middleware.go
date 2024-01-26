package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// https://twin.sh/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
var (
	reset  = "\033[0m"
	bold   = "\033[1m"
	italic = "\033[3m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	purple = "\033[35m"
	cyan   = "\033[36m"
)

func printUser(user *gotgbot.User) string {
	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"%s%s%s",
			bold,
			red,
			user.FirstName,
		),
	)

	if user.LastName != "" {
		sb.WriteString(" ")
		sb.WriteString(user.LastName)
	}

	sb.WriteString(reset)

	if user.Username != "" {
		sb.WriteString(
			fmt.Sprintf(
				" %s(@%s)%s",
				red,
				user.Username,
				reset,
			),
		)
	}

	return sb.String()
}

func onMessage(msg *gotgbot.Message) string {
	var sb strings.Builder

	// Time
	var msgTime string
	if msg.EditDate != 0 {
		msgTime = utils.TimestampToTime(msg.EditDate).Format("15:04:05")
	} else {
		msgTime = utils.TimestampToTime(msg.Date).Format("15:04:05")
	}

	sb.WriteString(
		fmt.Sprintf(
			"%s[%v]",
			cyan,
			msgTime,
		),
	)

	// Chat Title
	if msg.Chat.Title != "" {
		sb.WriteString(
			fmt.Sprintf(
				" %s:",
				msg.Chat.Title,
			),
		)
	}

	sb.WriteString(reset)

	// Sender
	if msg.GetSender() != nil {
		sb.WriteString(
			fmt.Sprintf(
				" %s",
				printUser(msg.From),
			),
		)
	}

	// Begin message
	sb.WriteString(
		fmt.Sprintf(
			"%s >>> %s",
			cyan,
			reset,
		),
	)

	// Was edited
	if msg.EditDate != 0 {
		sb.WriteString(
			fmt.Sprintf(
				"%s(editiert) %s",
				green,
				reset,
			),
		)
	}

	// Forwards
	if msg.ForwardOrigin != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sWeitergeleitet von %s",
				green,
				reset,
			),
		)

		mergedMessageOrigin := msg.ForwardOrigin.MergeMessageOrigin()

		if mergedMessageOrigin.SenderUser != nil { // User is visible
			sb.WriteString(
				fmt.Sprintf(
					"%s: ",
					printUser(mergedMessageOrigin.SenderUser),
				),
			)
		} else if mergedMessageOrigin.SenderUserName != "" { // User disallows linking to their profile on forwarding
			sb.WriteString(
				fmt.Sprintf(
					"%s%s%s:%s ",
					bold,
					red,
					mergedMessageOrigin.SenderUserName,
					reset,
				),
			)
		} else if mergedMessageOrigin.SenderChat != nil { // Message was originally sent on behalf of a group chat
			sb.WriteString(
				fmt.Sprintf(
					"%s%s%s",
					bold,
					red,
					mergedMessageOrigin.SenderChat.Title,
				),
			)

			if mergedMessageOrigin.AuthorSignature != "" {
				sb.WriteString(
					fmt.Sprintf(
						" (signiert von %s)",
						mergedMessageOrigin.AuthorSignature,
					),
				)
			}

			sb.WriteString(
				fmt.Sprintf(
					":%s ",
					reset,
				),
			)
		} else if mergedMessageOrigin.Chat != nil { // Message was originally sent to a channel
			sb.WriteString(
				fmt.Sprintf(
					"%s%s%s",
					bold,
					red,
					mergedMessageOrigin.Chat.Title,
				),
			)

			if mergedMessageOrigin.AuthorSignature != "" {
				sb.WriteString(
					fmt.Sprintf(
						" (signiert von %s)",
						mergedMessageOrigin.AuthorSignature,
					),
				)
			}

			sb.WriteString(
				fmt.Sprintf(
					":%s ",
					reset,
				),
			)
		}

	}

	// Reply
	if msg.ReplyToMessage != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sAntwort an %s%s: ",
				green,
				reset,
				printUser(msg.ReplyToMessage.From),
			),
		)
	}

	// External Replys
	if msg.ExternalReply != nil {
		// TODO
		sb.WriteString(
			fmt.Sprintf(
				"%sExterne Antwort%s: ",
				green,
				reset,
			),
		)
	}

	// Via bot
	if msg.ViaBot != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%svia %s%s: ",
				green,
				reset,
				printUser(msg.ViaBot),
			),
		)
	}

	// Files, etc.
	if msg.Animation != nil { // Animation: https://core.telegram.org/bots/api#animation
		sb.WriteString(
			fmt.Sprintf(
				"%s[GIF",
				purple,
			),
		)

		if msg.Animation.FileName != "" {
			sb.WriteString(
				fmt.Sprintf(
					": '%s'",
					msg.Animation.FileName,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Audio != nil { // Audio: https://core.telegram.org/bots/api#audio
		sb.WriteString(
			fmt.Sprintf(
				"%s[Audio",
				purple,
			),
		)

		if msg.Audio.Title != "" {
			sb.WriteString(
				fmt.Sprintf(
					": '%s'",
					msg.Audio.Title,
				),
			)
		}

		if msg.Audio.Performer != "" {
			if msg.Audio.Title == "" {
				sb.WriteString(": Unbekannt")
			}
			sb.WriteString(
				fmt.Sprintf(
					" von '%s'",
					msg.Audio.Performer,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Contact != nil { // Contact: https://core.telegram.org/bots/api#contact
		sb.WriteString(
			fmt.Sprintf(
				"%s[Kontakt: '%s",
				purple,
				msg.Contact.FirstName,
			),
		)

		if msg.Contact.LastName != "" {
			sb.WriteString(
				fmt.Sprintf(
					" %s",
					msg.Contact.LastName,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"', +%s",
				msg.Contact.PhoneNumber,
			),
		)

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Dice != nil { // Dice: https://core.telegram.org/bots/api#dice
		sb.WriteString(
			fmt.Sprintf(
				"%s[Zufallszahl: '%s' - '%d']%s ",
				purple,
				msg.Dice.Emoji,
				msg.Dice.Value,
				reset,
			),
		)
	} else if msg.Document != nil { // Document: https://core.telegram.org/bots/api#document
		sb.WriteString(
			fmt.Sprintf(
				"%s[Datei",
				purple,
			),
		)

		if msg.Document.FileName != "" {
			sb.WriteString(
				fmt.Sprintf(
					": '%s'",
					msg.Document.FileName,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Game != nil { // Game: https://core.telegram.org/bots/api#game
		sb.WriteString(
			fmt.Sprintf(
				"%s[Spiel: '%s' - '%s']%s ",
				purple,
				msg.Game.Title,
				msg.Game.Description,
				reset,
			),
		)
	} else if msg.Location != nil && msg.Venue == nil { // Location: https://core.telegram.org/bots/api#location
		sb.WriteString(
			fmt.Sprintf(
				"%s[Standort: '%f' Länge - '%f' Breite]%s ",
				purple,
				msg.Location.Longitude,
				msg.Location.Latitude,
				reset,
			),
		)
	} else if msg.Photo != nil && len(msg.Photo) > 0 { // Photo: https://core.telegram.org/bots/api#photosize
		bestResolutionPhoto := tgUtils.GetBestResolution(msg.Photo)
		sb.WriteString(
			fmt.Sprintf(
				"%s[Foto: %dx%d px]%s ",
				purple,
				bestResolutionPhoto.Width,
				bestResolutionPhoto.Height,
				reset,
			),
		)
	} else if msg.Sticker != nil { // Sticker: https://core.telegram.org/bots/api#sticker
		sb.WriteString(
			fmt.Sprintf(
				"%s[",
				purple,
			),
		)

		if msg.Sticker.IsAnimated {
			sb.WriteString("Animierter ")
		}

		sb.WriteString("Sticker")

		if msg.Sticker.Emoji != "" && msg.Sticker.SetName != "" {
			sb.WriteString(
				fmt.Sprintf(
					": '%s' aus '%s'",
					msg.Sticker.Emoji,
					msg.Sticker.SetName,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Venue != nil { // Venue: https://core.telegram.org/bots/api#venue
		sb.WriteString(
			fmt.Sprintf(
				"%s[Ort: '%s' in '%s', '%f' Länge, '%f' Breite]%s ",
				purple,
				msg.Venue.Title,
				msg.Venue.Address,
				msg.Venue.Location.Longitude,
				msg.Venue.Location.Latitude,
				reset,
			),
		)
	} else if msg.Video != nil { // Video: https://core.telegram.org/bots/api#video
		sb.WriteString(
			fmt.Sprintf(
				"%s[Video: ",
				purple,
			),
		)

		if msg.Video.FileName != "" {
			sb.WriteString(
				fmt.Sprintf(
					"'%s', ",
					msg.Video.FileName,
				),
			)
		}

		sb.WriteString(
			fmt.Sprintf(
				"%dx%d px, %d Sekunde",
				msg.Video.Width,
				msg.Video.Height,
				msg.Video.Duration,
			),
		)

		if msg.Video.Duration == 0 || msg.Video.Duration > 1 {
			sb.WriteString("n")
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.Voice != nil { // Voice: https://core.telegram.org/bots/api#voice
		sb.WriteString(
			fmt.Sprintf(
				"%s[Sprachnachricht: %d Sekunde",
				purple,
				msg.Voice.Duration,
			),
		)

		if msg.Voice.Duration == 0 || msg.Voice.Duration > 1 {
			sb.WriteString("n")
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	} else if msg.VideoNote != nil { // Video Note: https://core.telegram.org/bots/api#videonote
		sb.WriteString(
			fmt.Sprintf(
				"%s[Videonachricht: %d Sekunde",
				purple,
				msg.VideoNote.Duration,
			),
		)

		if msg.VideoNote.Duration == 0 || msg.VideoNote.Duration > 1 {
			sb.WriteString("n")
		}

		sb.WriteString(
			fmt.Sprintf(
				"]%s ",
				reset,
			),
		)
	}

	// Finally, the message text: https://core.telegram.org/bots/api#message
	if msg.Text != "" {
		sb.WriteString(msg.Text)
	}
	if msg.Caption != "" {
		sb.WriteString(msg.Caption)
	}

	// Service messages
	if msg.NewChatMembers != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sZur Gruppe hinzugefügt:%s ",
				yellow,
				reset,
			),
		)

		var newUsers []string
		for _, user := range msg.NewChatMembers {
			newUsers = append(newUsers, printUser(&user))
		}

		sb.WriteString(strings.Join(newUsers, ", "))
	}

	if msg.LeftChatMember != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sAus der Gruppe entfernt:%s %s",
				yellow,
				reset,
				printUser(msg.LeftChatMember),
			),
		)
	}

	if msg.NewChatTitle != "" {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppe umbenannt in '%s'%s",
				yellow,
				msg.NewChatTitle,
				reset,
			),
		)
	}

	if msg.NewChatPhoto != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppenbild geändert%s",
				yellow,
				reset,
			),
		)
	}

	if msg.DeleteChatPhoto {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppenbild entfernt%s",
				yellow,
				reset,
			),
		)
	}

	if msg.GroupChatCreated {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppe erstellt%s",
				yellow,
				reset,
			),
		)
	}

	if msg.SupergroupChatCreated {
		sb.WriteString(
			fmt.Sprintf(
				"%sSupergruppe erstellt%s",
				yellow,
				reset,
			),
		)
	}

	if msg.ChannelChatCreated {
		sb.WriteString(
			fmt.Sprintf(
				"%sKanal erstellt%s",
				yellow,
				reset,
			),
		)
	}

	if msg.PinnedMessage != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sNachricht angepinnt%s",
				yellow,
				reset,
			),
		)
	}

	return sb.String()
}

func onCallback(callback *gotgbot.CallbackQuery) string {
	var sb strings.Builder

	if callback.Message != nil {
		// Time
		if callback.Message.GetDate() != 0 {
			sb.WriteString(
				fmt.Sprintf(
					"%s[%v]%s",
					cyan,
					utils.TimestampToTime(callback.Message.GetDate()).Format("15:04:05"),
					reset,
				),
			)
		}

		// Chat Title
		if callback.Message.GetChat().Title != "" {
			if callback.Message.GetDate() != 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(
				fmt.Sprintf(
					"%s%s%s",
					cyan,
					callback.Message.GetChat().Title,
					reset,
				),
			)
		}

		if callback.Message.GetChat().Title != "" || callback.Message.GetDate() != 0 {
			sb.WriteString(
				fmt.Sprintf(
					"%s:%s ",
					cyan,
					reset,
				),
			)
		}
	}

	// Sender
	sb.WriteString(
		fmt.Sprintf(
			"%s",
			printUser(&callback.From),
		),
	)

	// Begin message
	sb.WriteString(
		fmt.Sprintf(
			"%s >>> %s%s(CallbackQuery)%s ",
			cyan,
			reset,
			green,
			reset,
		),
	)

	if callback.Data != "" {
		sb.WriteString(
			fmt.Sprintf(
				"%s%s%s",
				purple,
				callback.Data,
				reset,
			),
		)
	}

	return sb.String()
}

func onInlineQuery(query *gotgbot.InlineQuery) string {
	var sb strings.Builder

	// Time
	sb.WriteString(
		fmt.Sprintf(
			"%s[%v]%s ",
			cyan,
			time.Now().Format("15:04:05"),
			reset,
		),
	)

	if query.ChatType != "" {
		chatType := ""
		switch query.ChatType {
		case "channel":
			chatType = "In Kanal"
		case "group":
			chatType = "In Gruppe"
		case "supergroup":
			chatType = "In Supergruppe"
		case "private":
			chatType = "In Privatchat"
		case "sender":
			chatType = "In Bot-Chat"
		}
		sb.WriteString(
			fmt.Sprintf(
				"%s%s%s%s: ",
				italic,
				cyan,
				chatType,
				reset,
			),
		)
	}

	// Sender
	sb.WriteString(
		fmt.Sprintf(
			"%s",
			printUser(&query.From),
		),
	)

	// Begin message
	sb.WriteString(
		fmt.Sprintf(
			"%s >>> %s%s(InlineQuery)%s ",
			cyan,
			reset,
			green,
			reset,
		),
	)

	if query.Query != "" {
		sb.WriteString(
			fmt.Sprintf(
				"%s%s%s",
				purple,
				query.Query,
				reset,
			),
		)
	}

	return sb.String()
}

func PrintMessage(c *ext.Context) {
	var text string
	if c.Message != nil {
		text = onMessage(c.Message)
	} else if c.CallbackQuery != nil {
		text = onCallback(c.CallbackQuery)
	} else if c.InlineQuery != nil {
		text = onInlineQuery(c.InlineQuery)
	} else {
		text = fmt.Sprintf(
			"%s>>> %s%sUnbekannter Nachrichtentyp%s",
			cyan,
			reset,
			red,
			reset,
		)
	}

	println(text)
}

func OnError(err error) {
	lg := log.Err(err)
	lg.Send()
}
