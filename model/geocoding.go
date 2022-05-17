package model

import (
	"errors"

	"gopkg.in/telebot.v3"
)

var ErrAddressNotFound = errors.New("address not found")

type (
	GeocodingService interface {
		Geocode(address string) (telebot.Venue, error)
	}
)
