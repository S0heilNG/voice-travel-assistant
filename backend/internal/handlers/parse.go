package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"
	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/searchurl"
)

type parseRequest struct {
	Text string `json:"text"`
}

type parseResponse struct {
	Intent      nlu.Intent      `json:"intent"`
	Origin      *nlu.City       `json:"origin"`
	Destination *nlu.City       `json:"destination"`
	Date        *nlu.JalaliDate `json:"date"`
	Nights      int             `json:"nights"`
	Adults      int             `json:"adults"`
	Missing     []string        `json:"missing"`
	SearchURL   string          `json:"searchUrl"`
	// CitySupported is false when we understood a hotel/train/bus request but
	// 780.ir doesn't cover one of its cities, so the client can say so instead
	// of just showing an empty searchUrl. It stays true for flights and for
	// any request that isn't complete yet. (Renamed from hotelCitySupported now
	// that train and bus can also hit an unsupported city.)
	CitySupported bool `json:"citySupported"`
}

func parseText(c *fiber.Ctx) error {
	var req parseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	if strings.TrimSpace(req.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "\"text\" must not be empty"})
	}

	result := nlu.Parse(req.Text)

	missing := result.Missing
	if missing == nil {
		missing = []string{}
	}

	searchURL := ""
	citySupported := true
	if len(result.Missing) == 0 {
		switch result.Intent {
		case nlu.IntentFlightSearch:
			if u, err := searchurl.BuildFlightSearchURL(result); err == nil {
				searchURL = u
			}
		case nlu.IntentHotelSearch:
			// Missing is empty here, so Nights is set (>= 1).
			u, err := searchurl.BuildHotelSearchURL(result, result.Nights)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrHotelCityUnsupported):
				citySupported = false
			}
		case nlu.IntentTrainSearch:
			u, err := searchurl.BuildTrainSearchURL(result)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrTrainCityUnsupported):
				citySupported = false
			}
		case nlu.IntentBusSearch:
			u, err := searchurl.BuildBusSearchURL(result)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrBusCityUnsupported):
				citySupported = false
			}
		}
	}

	return c.JSON(parseResponse{
		Intent:        result.Intent,
		Origin:        result.Origin,
		Destination:   result.Destination,
		Date:          result.Date,
		Nights:        result.Nights,
		Adults:        result.Adults,
		Missing:       missing,
		SearchURL:     searchURL,
		CitySupported: citySupported,
	})
}
