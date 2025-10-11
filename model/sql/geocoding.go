package sql

import (
	"fmt"
	"net/url"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/utils/httpUtils"
)

type (
	geocodingService struct{}

	Location struct {
		PlaceId     int     `json:"place_id"`
		OsmType     string  `json:"osm_type"`
		OsmId       int     `json:"osm_id"`
		Lat         float64 `json:"lat,string"`
		Lng         float64 `json:"lon,string"`
		DisplayName string  `json:"display_name"`
		Category    string  `json:"category"`
		Type        string  `json:"type"`
	}
)

func NewGeocodingService() *geocodingService {
	return &geocodingService{}
}

func (db *geocodingService) Geocode(address string) (gotgbot.Venue, error) {
	requestUrl := url.URL{
		Scheme: "https",
		Host:   "nominatim.openstreetmap.org",
		Path:   "/search",
	}

	q := requestUrl.Query()
	q.Set("accept-language", "de")
	q.Set("limit", "1")
	q.Set("format", "jsonv2")
	q.Set("q", address)

	requestUrl.RawQuery = q.Encode()

	var response []Location
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      requestUrl.String(),
		Headers:  map[string]string{"User-Agent": "Gobot for Telegram"},
		Response: &response,
	})

	if err != nil {
		return gotgbot.Venue{}, fmt.Errorf("error while geocoding: %w, url: %s", err, requestUrl.String())
	}

	if len(response) == 0 {
		return gotgbot.Venue{}, model.ErrAddressNotFound
	}

	return gotgbot.Venue{
		Title:   response[0].DisplayName,
		Address: response[0].DisplayName,
		Location: gotgbot.Location{
			Latitude:  response[0].Lat,
			Longitude: response[0].Lng,
		},
	}, nil
}
