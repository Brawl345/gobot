package models

import (
	"errors"

	"gopkg.in/telebot.v3"
)

var ErrHomeAddressNotSet = errors.New("home address not set")

type HomeService interface {
	GetHome(user *telebot.User) (telebot.Venue, error)
	SetHome(user *telebot.User, venue *telebot.Venue) error
	DeleteHome(user *telebot.User) error
}
