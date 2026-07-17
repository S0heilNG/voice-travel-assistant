// Package nlu is a rule-based (no LLM) Persian NLU module: it takes raw
// speech-to-text output and extracts an intent plus the entities needed to
// build a 780.ir search URL (origin/destination city, date, passenger count).
package nlu

import (
	"strings"
	"time"
)

type Intent string

const (
	IntentFlightSearch Intent = "flight_search"
	IntentHotelSearch  Intent = "hotel_search"
	IntentUnknown      Intent = "unknown"
)

// ParseResult is the structured outcome of parsing one utterance.
type ParseResult struct {
	Intent      Intent
	Origin      *City // stays nil for hotel search
	Destination *City
	Date        *JalaliDate
	Adults      int
	RawText     string
	Missing     []string // e.g. []string{"destination", "date"}
}

// Parse extracts intent and entities from raw Persian text.
func Parse(text string) ParseResult {
	return parse(text, time.Now())
}

// parse is the testable core of Parse — it takes `now` explicitly so tests
// get deterministic date results instead of depending on the real clock.
func parse(text string, now time.Time) ParseResult {
	normalized := Normalize(text)
	intent := detectIntent(normalized)

	result := ParseResult{
		Intent:  intent,
		Adults:  1,
		RawText: text,
	}

	if intent == IntentUnknown {
		return result
	}

	origin, destination := extractOriginDestination(normalized)
	if intent == IntentHotelSearch {
		origin = nil
	}
	result.Origin = origin
	result.Destination = destination

	if intent == IntentFlightSearch && origin == nil {
		result.Missing = append(result.Missing, "origin")
	}
	if destination == nil {
		result.Missing = append(result.Missing, "destination")
	}

	result.Date = ParseDate(normalized, now)
	if result.Date == nil {
		result.Missing = append(result.Missing, "date")
	}

	return result
}

// extractOriginDestination looks for the "[از] X به Y" pattern: any known
// city mentioned before "به" is the origin, any known city after "به" is
// the destination. Without "به", a single recognized city is treated as
// the destination only (origin stays unknown) — e.g. "هتل در مشهد" or
// "می‌خوام برم کیش".
func extractOriginDestination(text string) (origin *City, destination *City) {
	matches := findCitiesInText(text)
	if len(matches) == 0 {
		return nil, nil
	}

	beIdx := strings.Index(text, "به")
	if beIdx == -1 {
		c := matches[0].city
		return nil, &c
	}

	for _, m := range matches {
		if m.start < beIdx && origin == nil {
			c := m.city
			origin = &c
		} else if m.start > beIdx && destination == nil {
			c := m.city
			destination = &c
		}
	}

	if origin == nil && destination == nil {
		c := matches[0].city
		destination = &c
	}
	return origin, destination
}
