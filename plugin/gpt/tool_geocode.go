package gpt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Brawl345/gobot/model"
)

type GeocodeTool struct {
	geocodingService model.GeocodingService
	chatID           int64
}

func NewGeocodeTool(geocodingService model.GeocodingService, chatID int64) *GeocodeTool {
	return &GeocodeTool{geocodingService: geocodingService, chatID: chatID}
}

func (t *GeocodeTool) Definition() FunctionTool {
	return FunctionTool{
		Type:        "function",
		Name:        "geocode",
		Description: "Ermittelt die Koordinaten und den vollständigen Namen eines Ortes, einer Adresse oder eines Bauwerks über OpenStreetMap. Nutze es, bevor du eine Karte (<tg-map>) zeigst.",
		Parameters: FunctionParameters{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "Ort, Adresse oder Name, z. B. \"Brandenburger Tor, Berlin\"",
				},
			},
			Required:             []string{"query"},
			AdditionalProperties: false,
		},
		Strict: false,
	}
}

func (t *GeocodeTool) Execute(arguments string) (any, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", errors.New("query must not be empty")
	}
	log.Debug().
		Str("query", args.Query).
		Int64("chat_id", t.chatID).
		Msg("geocode tool call")

	venue, err := t.geocodingService.Geocode(args.Query)
	if errors.Is(err, model.ErrAddressNotFound) {
		return "No location found.", nil
	}
	if err != nil {
		return "", fmt.Errorf("geocoding failed: %w", err)
	}
	return fmt.Sprintf("Name: %s\nlat: %.6f\nlong: %.6f",
		stripCitationChars(venue.Address), venue.Location.Latitude, venue.Location.Longitude), nil
}

func (t *GeocodeTool) Emoji() string {
	return "📍"
}

func (t *GeocodeTool) MaxCalls() int {
	return 3
}
