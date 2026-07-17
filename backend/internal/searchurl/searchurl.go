// Package searchurl builds 780.ir search result URLs from a resolved NLU
// parse result. It's kept separate from internal/nlu on purpose: nlu is
// about understanding Persian text (language in, structured intent out),
// while this package is about one specific downstream integration (780.ir's
// URL scheme). Keeping them apart means nlu doesn't need to know anything
// about 780.ir, and a future second consumer of nlu.ParseResult (or a
// second travel platform) wouldn't have to fight this package's assumptions.
package searchurl

import (
	"fmt"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"
)

// BuildFlightSearchURL builds a 780.ir flight search results URL from a
// parse result. Callers should check that result.Missing is empty and
// result.Intent is flight_search before calling this — it returns an error
// if intent, origin, destination, or date aren't all resolved.
func BuildFlightSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentFlightSearch {
		return "", fmt.Errorf("cannot build flight search URL: intent is %q, not %q", result.Intent, nlu.IntentFlightSearch)
	}
	if result.Origin == nil || result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build flight search URL: origin, destination, and date must all be resolved")
	}

	return fmt.Sprintf(
		"https://780.ir/tourism/flights/%s-%s?adult=%d&child=0&infant=0&departureDate=%s&sort=lowPrice",
		result.Origin.IATA, result.Destination.IATA, result.Adults, result.Date.String(),
	), nil
}
