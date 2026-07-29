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
	// ContextIntent is what the conversation was about before this utterance,
	// so a bare reply ("فردا") can be understood by a stateless parser. It is
	// only a default: text that names a service outright switches to it, and
	// the response's Intent is what actually happened.
	ContextIntent nlu.Intent `json:"contextIntent"`
	// ContextOrigin/ContextDestination are the cities already established in
	// the conversation, by canonical name. They exist so that switching
	// service can be checked against the city the user already gave: "قطار" on
	// its own names no city, but if the flow was about کیش we must answer that
	// Kish has no train rather than asking for a date first. Slot merging
	// stays in the client; these are only used to answer that question.
	ContextOrigin      string `json:"contextOrigin"`
	ContextDestination string `json:"contextDestination"`
}

type parseResponse struct {
	Intent      nlu.Intent      `json:"intent"`
	Origin      *nlu.City       `json:"origin"`
	Destination *nlu.City       `json:"destination"`
	Date        *nlu.JalaliDate `json:"date"`
	// ReturnDate is non-null only for a round-trip international flight. The
	// frontend needs it both to show the return leg on the confirm card and to
	// build the URL with tripMode=2.
	ReturnDate *nlu.JalaliDate `json:"returnDate"`
	Nights     int             `json:"nights"`
	Adults     int             `json:"adults"`
	Missing    []string        `json:"missing"`
	SearchURL  string          `json:"searchUrl"`
	// CitySupported is false when we understood a hotel/train/bus request but
	// 780.ir doesn't cover one of its cities, so the client can say so instead
	// of just showing an empty searchUrl. It stays true for flights and for
	// any request that isn't complete yet. (Renamed from hotelCitySupported now
	// that train and bus can also hit an unsupported city.)
	CitySupported bool `json:"citySupported"`
	// Alternatives lists the services 780.ir *can* search for this city when
	// the requested one can't be. It exists so a dead end can offer a way
	// onward instead of just refusing. Computed server-side because only the
	// backend holds every per-service table — duplicating them client-side
	// would be one more thing to drift.
	Alternatives []nlu.Intent `json:"alternatives"`
	// UnsupportedService names a product 780.ir has no standalone version of
	// (currently only villas/ecolodges). It does not affect Intent — a valid
	// request must never be stolen — but the client needs it to explain the one
	// dead end that isn't about a city.
	UnsupportedService string `json:"unsupportedService"`
}

func (a *api) parseText(c *fiber.Ctx) error {
	var req parseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	if strings.TrimSpace(req.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "\"text\" must not be empty"})
	}

	result := nlu.ParseWithContext(req.Text, req.ContextIntent)

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
		case nlu.IntentIntlFlightSearch:
			u, err := searchurl.BuildInternationalFlightSearchURL(result)
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
		case nlu.IntentTourSearch:
			u, err := searchurl.BuildTourSearchURL(result)
			switch {
			case err == nil:
				searchURL = u
			case errors.Is(err, searchurl.ErrTourDestUnsupported):
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

	// What else could this city be searched for? Only worth answering once a
	// city is actually resolved; which endpoint to describe depends on which
	// one failed, and for route services the origin is the culprit as often as
	// the destination.
	alternatives := alternativesFor(withContextCities(result, req))

	// Villas/ecolodges are the one dead end that isn't about a city. Reported
	// separately from Intent so it can be explained without ever overriding a
	// request we can actually serve.
	unsupportedService := nlu.DetectUnsupportedService(req.Text)

	// Record the interaction (no-op if logging is disabled; never fails the
	// request). Categorized so we can group the "couldn't help" cases.
	category, detail := categorize(req.Text, result, citySupported)
	a.store.LogParse(analytics.ParseLog{
		SessionID:            sessionID(c),
		RawText:              req.Text,
		Intent:               string(result.Intent),
		Origin:               cityName(result.Origin),
		Destination:          cityName(result.Destination),
		Date:                 dateString(result.Date),
		Nights:               result.Nights,
		Missing:              strings.Join(missing, ","),
		CitySupported:        citySupported,
		URLFromThisUtterance: searchURL != "",
		Category:             category,
		CategoryDetail:       detail,
	})

	return c.JSON(parseResponse{
		Intent:             result.Intent,
		Origin:             result.Origin,
		Destination:        result.Destination,
		Date:               result.Date,
		ReturnDate:         result.ReturnDate,
		Nights:             result.Nights,
		Adults:             result.Adults,
		Missing:            missing,
		SearchURL:          searchURL,
		CitySupported:      citySupported,
		Alternatives:       alternatives,
		UnsupportedService: unsupportedService,
	})
}

// withContextCities fills in cities this utterance didn't name from the ones
// the conversation already established.
//
// It is used ONLY to decide whether to offer alternatives, never to build a
// search URL or to answer the client — slot merging belongs to the client,
// which owns the conversation state. Without it, switching service on a bare
// "قطار" would look city-less and we'd ask for a date before discovering the
// carried-over city has no train.
func withContextCities(result nlu.ParseResult, req parseRequest) nlu.ParseResult {
	if result.Origin == nil {
		result.Origin = nlu.LookupCity(req.ContextOrigin)
	}
	if result.Destination == nil {
		result.Destination = nlu.LookupCity(req.ContextDestination)
	}
	return result
}

// alternativesFor returns the other services that could search this request's
// city, or nil when there's nothing to offer.
//
// It deliberately does NOT wait for the request to be complete. A city that
// this service can't serve is a dead end the moment we recognize it, and
// telling the user immediately is the whole point — asking "from where?" first
// and only then refusing wastes the turn we were trying to save.
//
// For route services either end can be the culprit, so both are checked.
func alternativesFor(result nlu.ParseResult) []nlu.Intent {
	for _, c := range []*nlu.City{result.Destination, result.Origin} {
		if c == nil || searchurl.CitySupportedFor(result.Intent, *c) {
			continue
		}
		if alts := searchurl.AlternativeServices(*c, result.Intent); len(alts) > 0 {
			return alts
		}
	}
	return nil
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
func categorize(text string, result nlu.ParseResult, citySupported bool) (category, detail string) {
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
