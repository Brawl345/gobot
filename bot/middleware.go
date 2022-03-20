package bot

import (
	"fmt"
	"gopkg.in/telebot.v3"
	"log"
	"strings"
)

// https://twin.sh/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
var (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	purple = "\033[35m"
	cyan   = "\033[36m"
	gray   = "\033[37m"
	white  = "\033[97m"
)

func printUser(user *telebot.User) string {
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

func onMessage(msg *telebot.Message) string {
	var sb strings.Builder

	// Time
	var msgTime string
	if msg.LastEdit != 0 {
		msgTime = msg.LastEdited().Format("15:04:05")
	} else {
		msgTime = msg.Time().Format("15:04:05")
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
	if msg.Sender != nil {
		sb.WriteString(
			fmt.Sprintf(
				" %s",
				printUser(msg.Sender),
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
	if msg.LastEdit != 0 {
		sb.WriteString(
			fmt.Sprintf(
				"%s(editiert) %s",
				green,
				reset,
			),
		)
	}

	// Forwards
	if msg.IsForwarded() || msg.OriginalSenderName != "" {
		sb.WriteString(
			fmt.Sprintf(
				"%sWeitergeleitet von ",
				green,
			),
		)

		if msg.OriginalSender != nil {
			sb.WriteString(
				fmt.Sprintf(
					"%s: ",
					printUser(msg.OriginalSender),
				),
			)
		} else { // User disallows linking to their profile on forwarding
			sb.WriteString(
				fmt.Sprintf(
					"%s%s%s:%s ",
					bold,
					red,
					msg.OriginalSenderName,
					reset,
				),
			)
		}
	}

	// Reply
	if msg.IsReply() {
		sb.WriteString(
			fmt.Sprintf(
				"%sAntwort an %s%s: ",
				green,
				reset,
				printUser(msg.ReplyTo.Sender),
			),
		)
	}

	// Via bot
	if msg.Via != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%svia %s%s: ",
				green,
				reset,
				printUser(msg.Via),
			),
		)
	}

	// Files, etc.

	// Animation: https://core.telegram.org/bots/api#animation
	if msg.Animation != nil {
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
	}

	// Audio: https://core.telegram.org/bots/api#audio
	if msg.Audio != nil {
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
	}

	// Contact: https://core.telegram.org/bots/api#contact
	if msg.Contact != nil {
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
	}

	// Dice: https://core.telegram.org/bots/api#dice
	if msg.Dice != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[Zufallszahl: '%s' - '%d']%s ",
				purple,
				msg.Dice.Type,
				msg.Dice.Value,
				reset,
			),
		)
	}

	// Document: https://core.telegram.org/bots/api#document
	if msg.Document != nil {
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
	}

	// Game: https://core.telegram.org/bots/api#game
	if msg.Game != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[Spiel: '%s' - '%s']%s ",
				purple,
				msg.Game.Title,
				msg.Game.Description,
				reset,
			),
		)
	}

	// Location: https://core.telegram.org/bots/api#location
	if msg.Location != nil && msg.Venue == nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[Standort: '%f' Länge - '%f' Breite]%s ",
				purple,
				msg.Location.Lng,
				msg.Location.Lat,
				reset,
			),
		)
	}

	// Photo: https://core.telegram.org/bots/api#photosize
	if msg.Photo != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[Foto: %dx%d px]%s ",
				purple,
				msg.Photo.Width,
				msg.Photo.Height,
				reset,
			),
		)
	}

	// Sticker: https://core.telegram.org/bots/api#sticker
	if msg.Sticker != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[",
				purple,
			),
		)

		if msg.Sticker.Animated {
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
	}

	// Venue: https://core.telegram.org/bots/api#venue
	if msg.Venue != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%s[Ort: '%s' in '%s', '%f' Länge, '%f' Breite]%s ",
				purple,
				msg.Venue.Title,
				msg.Venue.Address,
				msg.Venue.Location.Lng,
				msg.Venue.Location.Lat,
				reset,
			),
		)
	}

	// Video: https://core.telegram.org/bots/api#video
	if msg.Video != nil {
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
	}

	// Voice: https://core.telegram.org/bots/api#voice
	if msg.Voice != nil {
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
	}

	// Video Note: https://core.telegram.org/bots/api#videonote
	if msg.VideoNote != nil {
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
	if msg.UsersJoined != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sZur Gruppe hinzugefügt:%s ",
				yellow,
				reset,
			),
		)

		var newUsers []string
		for _, user := range msg.UsersJoined {
			newUsers = append(newUsers, printUser(&user))
		}

		sb.WriteString(strings.Join(newUsers, ", "))
	}

	if msg.UserLeft != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sAus der Gruppe entfernt:%s %s",
				yellow,
				reset,
				printUser(msg.UserLeft),
			),
		)
	}

	if msg.NewGroupTitle != "" {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppe umbenannt in '%s'%s",
				yellow,
				msg.NewGroupTitle,
				reset,
			),
		)
	}

	if msg.NewGroupPhoto != nil {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppenbild geändert%s",
				yellow,
				reset,
			),
		)
	}

	if msg.GroupPhotoDeleted {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppenbild entfernt%s",
				yellow,
				reset,
			),
		)
	}

	if msg.GroupCreated {
		sb.WriteString(
			fmt.Sprintf(
				"%sGruppe erstellt%s",
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

func PrintMessage(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {

		var text string
		if c.Message() != nil {
			text = onMessage(c.Message())
		}

		log.Println(text)
		return next(c)
	}
}
