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
	IntentTrainSearch  Intent = "train_search"
	IntentBusSearch    Intent = "bus_search"
	// IntentHelp covers "what can you do?" and bare greetings — the user isn't
	// searching yet, so instead of dead-ending on the error screen we show a
	// friendly capabilities message. It carries no slots.
	IntentHelp    Intent = "help"
	IntentUnknown Intent = "unknown"
)

// isRouteIntent reports whether the intent describes an origin→destination
// trip (flight, train, bus) as opposed to a stay (hotel). Route intents all
// need an origin, a destination, and a single date.
func isRouteIntent(i Intent) bool {
	return i == IntentFlightSearch || i == IntentTrainSearch || i == IntentBusSearch
}

// ParseResult is the structured outcome of parsing one utterance.
type ParseResult struct {
	Intent      Intent
	Origin      *City // stays nil for hotel search
	Destination *City
	Date        *JalaliDate // check-in date for hotels, departure date for flights
	Nights      int         // hotel stay length; 0 when unset/not applicable
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

	// Help and unknown carry no entities, so there's nothing more to extract.
	if intent == IntentUnknown || intent == IntentHelp {
		return result
	}

	origin, destination := extractOriginDestination(normalized, intent)
	result.Origin = origin
	result.Destination = destination

	if isRouteIntent(intent) && origin == nil {
		result.Missing = append(result.Missing, "origin")
	}
	if destination == nil {
		result.Missing = append(result.Missing, "destination")
	}

	if intent == IntentHotelSearch {
		// Hotels need a check-in date and a stay length (from a range or an
		// explicit nights count). Nights defaults to nothing so we ask.
		checkIn, nights := parseHotelStay(normalized, now)
		result.Date = checkIn
		result.Nights = nights
		if checkIn == nil {
			result.Missing = append(result.Missing, "date")
		}
		if nights == 0 {
			result.Missing = append(result.Missing, "nights")
		}
	} else {
		result.Date = ParseDate(normalized, now)
		if result.Date == nil {
			result.Missing = append(result.Missing, "date")
		}
	}

	return result
}

// maxAzWordGap bounds how far a city name may be from an "از" (from) marker
// for that marker to count as identifying the origin — "از" alone is a very
// common word ("یکی از بهترین هتل‌ها") and must not be treated as an origin
// marker unless a known city actually follows it closely.
const maxAzWordGap = 3

// extractOriginDestination resolves origin/destination city entities from
// normalized text.
//
// For hotel search, only a single destination city ever matters: the first
// recognized city in the text, origin always nil (e.g. "هتل مشهد تهران"
// means "a hotel in Mashhad, [also/or] Tehran", not a route — so we don't
// try to split it into origin/destination the way we do for flights).
//
// For flight search, the rules are applied in this priority order:
//  1. "از X" (with X a known city within maxAzWordGap words of "از") → X is
//     the origin. If "به Y" is also present, Y is the destination; otherwise
//     the destination is whichever other recognized city appears anywhere
//     else in the text (this correctly handles reversed phrasing like
//     "می‌خوام برم مشهد از تهران", where the destination is mentioned before
//     the "از" origin marker).
//  2. No "از" but "به Y" is present → any city before "به" is the origin,
//     Y (after "به") is the destination.
//  3. Neither marker: a single recognized city is the destination only
//     (origin unknown); two or more cities are read as a bare route
//     shorthand ("تهران مشهد") — first mentioned is origin, second is
//     destination.
func extractOriginDestination(text string, intent Intent) (origin *City, destination *City) {
	matches := findCitiesInText(text)
	if len(matches) == 0 {
		return nil, nil
	}

	if intent == IntentHotelSearch {
		c := matches[0].city
		return nil, &c
	}

	beIdx := standaloneWordIndex(text, "به")

	if azMatch := findOriginMarker(text, matches); azMatch != nil {
		originCity := azMatch.city
		origin = &originCity

		if beIdx != -1 {
			for _, m := range matches {
				if m.start > beIdx {
					c := m.city
					destination = &c
					break
				}
			}
		}
		if destination == nil {
			for _, m := range matches {
				if m.start != azMatch.start {
					c := m.city
					destination = &c
					break
				}
			}
		}
		return origin, destination
	}

	if beIdx != -1 {
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

	if len(matches) == 1 {
		c := matches[0].city
		return nil, &c
	}
	o := matches[0].city
	d := matches[1].city
	return &o, &d
}

// standaloneWordIndex returns the byte offset of `target` as a whole word
// in text (space-separated, as produced by Normalize), or -1 if it only
// appears as a substring of some other word. This matters because several
// weekday names ("پنجشنبه", "چهارشنبه", ...) end in "به" — a plain
// strings.Index("به") would wrongly match inside them.
func standaloneWordIndex(text, target string) int {
	offset := 0
	for _, w := range strings.Split(text, " ") {
		if w == target {
			return offset
		}
		offset += len(w) + 1
	}
	return -1
}

// findOriginMarker looks for a standalone "از" word (not a substring inside
// another word, e.g. "اندازه") that has a recognized city within
// maxAzWordGap words after it, and returns that city match.
func findOriginMarker(text string, matches []cityMatch) *cityMatch {
	words := strings.Split(text, " ")
	wordStart := make([]int, len(words))
	offset := 0
	for i, w := range words {
		wordStart[i] = offset
		offset += len(w) + 1
	}

	wordIndexAt := func(bytePos int) int {
		idx := 0
		for i, start := range wordStart {
			if start <= bytePos {
				idx = i
			} else {
				break
			}
		}
		return idx
	}

	for i, w := range words {
		if w != "از" {
			continue
		}
		for _, m := range matches {
			if m.start <= wordStart[i] {
				continue
			}
			if wordIndexAt(m.start)-i <= maxAzWordGap {
				mm := m
				return &mm
			}
		}
	}
	return nil
}
