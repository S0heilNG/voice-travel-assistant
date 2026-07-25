package nlu

import "strings"

// Mode keywords. "بلیط"/"بلیت" (ticket) are deliberately NOT here: a ticket
// word is generic — "بلیط قطار" is a *train* ticket — so it must not by itself
// decide the mode, or it would hijack "بلیط قطار ..." into a flight. Ticket
// only implies flight as a last resort (see detectIntent).
var trainKeywords = []string{"قطار"}
var busKeywords = []string{"اتوبوس"}
var flightKeywords = []string{"پرواز", "هواپیما"}
var hotelKeywords = []string{"هتل", "اقامتگاه", "اتاق", "جا برای موندن"}

// Short/risky keywords matched as whole words only: "ترن" is a substring of
// "اینترنت" and "بوس" of "اتوبوس"/"بوسه", so a plain substring search would
// misfire (same lesson as "به" inside "پنجشنبه").
var trainWordKeywords = []string{"ترن"}
var busWordKeywords = []string{"بوس"}

var ticketKeywords = []string{"بلیط", "بلیت"}

// detectIntent classifies normalized text into one of the search modes.
//
// The mode is chosen by the earliest *specific* mode keyword in the sentence
// (train, bus, flight, or hotel). Crucially, train and bus keywords win over a
// generic ticket word: "بلیط قطار تهران مشهد" is a train search, not a flight
// one, even though "بلیط" appears first.
//
// Fallback: with no specific mode keyword, a generic ticket word or any known
// city name means flight search — the default travel intent for this MVP
// (e.g. "می‌خوام برم کیش").
func detectIntent(text string) Intent {
	trainIdx := minIndex(firstKeywordIndex(text, trainKeywords), firstWholeWordIndex(text, trainWordKeywords))
	busIdx := minIndex(firstKeywordIndex(text, busKeywords), firstWholeWordIndex(text, busWordKeywords))
	flightIdx := firstKeywordIndex(text, flightKeywords)
	hotelIdx := firstKeywordIndex(text, hotelKeywords)

	best, bestIdx := IntentUnknown, -1
	for _, c := range []struct {
		intent Intent
		idx    int
	}{
		{IntentTrainSearch, trainIdx},
		{IntentBusSearch, busIdx},
		{IntentFlightSearch, flightIdx},
		{IntentHotelSearch, hotelIdx},
	} {
		if c.idx != -1 && (bestIdx == -1 || c.idx < bestIdx) {
			best, bestIdx = c.intent, c.idx
		}
	}
	if bestIdx != -1 {
		return best
	}

	if firstKeywordIndex(text, ticketKeywords) != -1 || len(findCitiesInText(text)) > 0 {
		return IntentFlightSearch
	}
	return IntentUnknown
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

// firstWholeWordIndex finds the earliest keyword that appears as a standalone
// space-separated word (not as a substring of a larger word).
func firstWholeWordIndex(text string, keywords []string) int {
	best := -1
	for _, kw := range keywords {
		if idx := standaloneWordIndex(text, kw); idx != -1 && (best == -1 || idx < best) {
			best = idx
		}
	}
	return best
}

// minIndex returns the smaller of two indices, treating -1 (not found) as
// "no value" rather than the smallest number.
func minIndex(a, b int) int {
	switch {
	case a == -1:
		return b
	case b == -1:
		return a
	case a < b:
		return a
	default:
		return b
	}
}
