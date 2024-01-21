package model

import (
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var ErrAddressNotFound = errors.New("address not found")

type (
	GeocodingService interface {
		Geocode(address string) (gotgbot.Venue, error)
	}
)
