package handlers

import (
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
	if result.Intent == nlu.IntentFlightSearch && len(result.Missing) == 0 {
		if u, err := searchurl.BuildFlightSearchURL(result); err == nil {
			searchURL = u
		}
	}

	return c.JSON(parseResponse{
		Intent:      result.Intent,
		Origin:      result.Origin,
		Destination: result.Destination,
		Date:        result.Date,
		Adults:      result.Adults,
		Missing:     missing,
		SearchURL:   searchURL,
	})
}
