package sql

import (
	"fmt"
	"net/url"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"gopkg.in/telebot.v3"
)

type (
	geocodingService struct{}

	Location struct {
		PlaceId     int     `json:"place_id"`
		OsmType     string  `json:"osm_type"`
		OsmId       int     `json:"osm_id"`
		Lat         float32 `json:"lat,string"`
		Lng         float32 `json:"lon,string"`
		DisplayName string  `json:"display_name"`
		Category    string  `json:"category"`
		Type        string  `json:"type"`
	}
)

func NewGeocodingService() *geocodingService {
	return &geocodingService{}
}

func (db *geocodingService) Geocode(address string) (telebot.Venue, error) {
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "nominatim.openstreetmap.org",
		Path:   "/search.php",
	}

	q := requestUrl.Query()
	q.Set("accept-language", "de")
	q.Set("limit", "1")
	q.Set("format", "jsonv2")
	q.Set("q", address)

	requestUrl.RawQuery = q.Encode()

	var response []Location
	err := httpUtils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"User-Agent": "Gobot for Telegram",
		},
		&response,
	)

	if err != nil {
		return telebot.Venue{}, fmt.Errorf("error while geocoding: %w, url: %s", err, requestUrl.String())
	}

	if len(response) == 0 {
		return telebot.Venue{}, model.ErrAddressNotFound
	}

	return telebot.Venue{
		Title:   response[0].DisplayName,
		Address: response[0].DisplayName,
		Location: telebot.Location{
			Lat: response[0].Lat,
			Lng: response[0].Lng,
		},
	}, nil
}
