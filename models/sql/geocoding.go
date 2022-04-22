package sql

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/utils"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	geocodingService struct {
		*sqlx.DB
	}

	Location struct {
		PlaceId     int     `json:"place_id"`
		OsmType     string  `json:"osm_type"`
		OsmId       int     `json:"osm_id"`
		Lat         float32 `json:"lat,string"` // this hurts but telebot doesn't uses float32
		Lng         float32 `json:"lon,string"`
		DisplayName string  `json:"display_name"`
		Category    string  `json:"category"`
		Type        string  `json:"type"`
	}

	Geocoding struct {
		Address string  `db:"address"`
		Lat     float32 `db:"latitude"`
		Lng     float32 `db:"longitude"`
	}
)

func NewGeocodingService(db *sqlx.DB) *geocodingService {
	return &geocodingService{db}
}

func doAPIRequest(address string) (*telebot.Venue, error) {
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
	err := utils.GetRequestWithHeader(
		requestUrl.String(),
		map[string]string{
			"User-Agent": "Gobot for Telegram",
		},
		&response,
	)

	if err != nil {
		return nil, fmt.Errorf("error while geocoding: %w, url: %s", err, requestUrl.String())
	}

	if len(response) == 0 {
		return nil, nil
	}

	return &telebot.Venue{
		Title:   response[0].DisplayName,
		Address: response[0].DisplayName,
		Location: telebot.Location{
			Lat: response[0].Lat,
			Lng: response[0].Lng,
		},
	}, nil
}

func (db *geocodingService) Geocode(address string) (*telebot.Venue, error) {
	address = strings.ToLower(address)
	const getQuery = `SELECT address, latitude, longitude
	FROM geocoding
	RIGHT JOIN geocoding_queries gq ON geocoding.id = gq.geocoding_id
	WHERE gq.query = ?`

	var geocoding Geocoding
	err := db.Get(&geocoding, getQuery, address)
	if err == nil {
		return &telebot.Venue{
			Title:   geocoding.Address,
			Address: geocoding.Address,
			Location: telebot.Location{
				Lat: geocoding.Lat,
				Lng: geocoding.Lng,
			},
		}, nil
	}

	venue, err := doAPIRequest(address)
	if err != nil {
		return nil, err
	}

	if venue == nil {
		return nil, models.ErrAddressNotFound
	}

	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const insertAddressQuery = `INSERT INTO geocoding 
    (address, latitude, longitude) 
	VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`
	res, err := tx.Exec(insertAddressQuery, venue.Address, venue.Location.Lat, venue.Location.Lng)
	if err != nil {
		return nil, err
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	log.Print(lastInsertId)

	const insertQueryForAddressQuery = `INSERT INTO geocoding_queries
	(query, geocoding_id)
	VALUES (?, ?)`
	_, err = tx.Exec(insertQueryForAddressQuery, address, lastInsertId)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return venue, nil
}
