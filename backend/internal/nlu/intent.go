package nlu

import "strings"

var flightKeywords = []string{"بلیط", "پرواز", "هواپیما", "بلیت"}
var hotelKeywords = []string{"هتل", "اقامتگاه", "اتاق", "جا برای موندن"}

// detectIntent classifies normalized text as a flight search, hotel search,
// or unknown, based on keyword presence. If both flight and hotel keywords
// appear, whichever occurs first in the sentence wins.
//
// Fallback: if no keyword matches but a known city is mentioned, we assume
// flight search — the default travel intent for this MVP. e.g. "می‌خوام
// برم کیش" has no "بلیط"/"پرواز" but clearly means "I want to fly to Kish".
func detectIntent(text string) Intent {
	flightIdx := firstKeywordIndex(text, flightKeywords)
	hotelIdx := firstKeywordIndex(text, hotelKeywords)

	switch {
	case flightIdx == -1 && hotelIdx == -1:
		if len(findCitiesInText(text)) > 0 {
			return IntentFlightSearch
		}
		return IntentUnknown
	case hotelIdx == -1:
		return IntentFlightSearch
	case flightIdx == -1:
		return IntentHotelSearch
	case flightIdx <= hotelIdx:
		return IntentFlightSearch
	default:
		return IntentHotelSearch
	}
}

func firstKeywordIndex(text string, keywords []string) int {
	best := -1
	for _, kw := range keywords {
		if idx := strings.Index(text, kw); idx != -1 && (best == -1 || idx < best) {
			best = idx
		}
	}
	return best
}
