package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/analytics"
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

func (a *api) parseText(c *fiber.Ctx) error {
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
			u, err := searchurl.BuildFlightSearchURL(result)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrFlightCityUnsupported):
				citySupported = false
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

	// Record the interaction (no-op if logging is disabled; never fails the
	// request). Categorized so we can group the "couldn't help" cases.
	category, detail := categorize(req.Text, result, citySupported, searchURL != "")
	a.store.LogParse(analytics.ParseLog{
		SessionID:      sessionID(c),
		RawText:        req.Text,
		Intent:         string(result.Intent),
		Origin:         cityName(result.Origin),
		Destination:    cityName(result.Destination),
		Date:           dateString(result.Date),
		Nights:         result.Nights,
		Missing:        strings.Join(missing, ","),
		CitySupported:  citySupported,
		SearchURLBuilt: searchURL != "",
		Category:       category,
		CategoryDetail: detail,
	})

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

func cityName(c *nlu.City) string {
	if c == nil {
		return ""
	}
	return c.Name
}

func dateString(d *nlu.JalaliDate) string {
	if d == nil {
		return ""
	}
	return d.String()
}

// categorize groups a parse for analysis. The category is for grouping only —
// the raw text is always stored regardless. Empty category means the request
// was fully resolved (a success at the parse level; funnel events tell whether
// the user then went to 780).
func categorize(text string, result nlu.ParseResult, citySupported, urlBuilt bool) (category, detail string) {
	// A request for a service we don't offer is the most valuable "couldn't
	// help" signal, so it wins even when a city made us guess a real intent
	// (e.g. "تور کیش" parses as flight because کیش is a city).
	if svc := nlu.DetectUnsupportedService(text); svc != "" {
		return "unsupported_service", svc
	}
	switch result.Intent {
	case nlu.IntentHelp:
		return "help", ""
	case nlu.IntentUnknown:
		return "unknown_intent", ""
	}
	if !citySupported {
		return "unsupported_city_for_service", string(result.Intent)
	}
	if len(result.Missing) > 0 {
		return "incomplete", strings.Join(result.Missing, ",")
	}
	return "", ""
}
