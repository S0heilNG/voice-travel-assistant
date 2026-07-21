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
	Adults      int             `json:"adults"`
	Missing     []string        `json:"missing"`
	SearchURL   string          `json:"searchUrl"`
	// HotelCitySupported is false when we understood a hotel request for a
	// city 780.ir doesn't cover, so the client can say so instead of just
	// showing an empty searchUrl. It stays true for every non-hotel request.
	HotelCitySupported bool `json:"hotelCitySupported"`
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
	hotelCitySupported := true
	if len(result.Missing) == 0 {
		switch result.Intent {
		case nlu.IntentFlightSearch:
			if u, err := searchurl.BuildFlightSearchURL(result); err == nil {
				searchURL = u
			}
		case nlu.IntentHotelSearch:
			// One night is the temporary default until we ask how many.
			u, err := searchurl.BuildHotelSearchURL(result, 1)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrHotelCityUnsupported):
				hotelCitySupported = false
			}
		}
	}

	return c.JSON(parseResponse{
		Intent:             result.Intent,
		Origin:             result.Origin,
		Destination:        result.Destination,
		Date:               result.Date,
		Adults:             result.Adults,
		Missing:            missing,
		SearchURL:          searchURL,
		HotelCitySupported: hotelCitySupported,
	})
}
