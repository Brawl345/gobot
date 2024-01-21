package model

import (
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var ErrHomeAddressNotSet = errors.New("home address not set")

type HomeService interface {
	GetHome(user *gotgbot.User) (gotgbot.Venue, error)
	SetHome(user *gotgbot.User, venue *gotgbot.Venue) error
	DeleteHome(user *gotgbot.User) error
}
