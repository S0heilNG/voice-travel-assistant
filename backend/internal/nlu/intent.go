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

// tourWordKeywords is matched as a whole word only: "تور" is a substring of
// "دستور", "موتور", "کنسرتور" and plenty more.
//
// ⚠️ Accepted risk: even as a whole word, "تور" has non-travel senses — a
// volleyball net ("تور والیبال"), a bridal veil ("تور عروس"), a fishing net
// ("تور ماهیگیری"). Those parse as tour searches. This is the same trade-off
// already accepted for the bare-city fallback: the user knows they are talking
// to a travel assistant, so a sentence containing "تور" is nearly always about
// travel. Revisit if the product ever answers general questions. In practice
// the damage is bounded — with no known destination the assistant just asks
// "کدوم مقصد؟" rather than doing anything wrong.
var tourWordKeywords = []string{"تور"}

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
	tourIdx := firstWholeWordIndex(text, tourWordKeywords)

	best, bestIdx := IntentUnknown, -1
	for _, c := range []struct {
		intent Intent
		idx    int
	}{
		{IntentTrainSearch, trainIdx},
		{IntentBusSearch, busIdx},
		{IntentFlightSearch, flightIdx},
		{IntentHotelSearch, hotelIdx},
		{IntentTourSearch, tourIdx},
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

	// Lowest priority, only once we're sure there's no search here: a bare
	// greeting or a "what can you do?" question → show the help screen instead
	// of an error. Checked last so "سلام بلیط تهران مشهد" stays a flight search.
	if isHelpQuery(text) {
		return IntentHelp
	}
	return IntentUnknown
}

// internationalWordKeywords are matched as whole words: "خارجی" is a substring
// of nothing common, but whole-word matching is the house rule for this module
// and costs nothing here.
var internationalWordKeywords = []string{"خارجی", "خارج"}

// internationalPhraseKeywords are distinctive enough to match as substrings.
// Normalize turns the ZWNJ in "بین‌المللی" into a space, so the spaced form
// covers both spellings.
var internationalPhraseKeywords = []string{"بین المللی", "بینالمللی"}

// isInternationalFlight decides whether a flight search leaves Iran.
//
// The city is the primary signal, not the wording: "بلیط تهران به استانبول"
// must be international even though nothing in it says so. An explicit
// "خارجی"/"بین المللی" also forces it, which is what lets a user ask for an
// international flight before naming any city ("پرواز خارجی می‌خوام") and get
// asked for the route rather than being routed to the domestic search.
func isInternationalFlight(text string, origin, destination *City) bool {
	if firstWholeWordIndex(text, internationalWordKeywords) != -1 {
		return true
	}
	if firstKeywordIndex(text, internationalPhraseKeywords) != -1 {
		return true
	}
	return IsForeignCity(origin) || IsForeignCity(destination)
}

// roundTripWordKeywords / roundTripPhraseKeywords signal that the user wants a
// return leg. "برگشت" as a whole word covers "رفت و برگشت" too (Normalize has
// already split the ZWNJ), and "دوطرفه"/"دو طرفه" are the common alternatives.
var roundTripWordKeywords = []string{"برگشت", "برگردم", "برمیگردم", "دوطرفه"}
var roundTripPhraseKeywords = []string{"دو طرفه", "رفت و برگشت", "برمی گردم"}

// wantsRoundTrip reports whether the user asked for a return leg without
// necessarily giving its date. Used to decide between asking "when do you come
// back?" and silently defaulting to one-way.
func wantsRoundTrip(text string) bool {
	return firstWholeWordIndex(text, roundTripWordKeywords) != -1 ||
		firstKeywordIndex(text, roundTripPhraseKeywords) != -1
}

// greetingKeywords are matched as whole words: "سلام" is a substring of
// "سلامت"/"سلامتی" (health), which are not greetings.
var greetingKeywords = []string{"سلام", "درود", "سلم"}

// capabilityKeywords catch "what can you do?" / "help" phrasings. Matched as
// substrings (they're distinctive); they only reach this point when no service
// keyword or city is present, so "کمکم کن بلیط بخرم" stays a flight search.
var capabilityKeywords = []string{
	"چیکار", "چی کار", "چه کار", "چیکارا", "چه کمکی", "کمکم کن",
	"چی بلدی", "چیا بلدی", "بلدی", "راهنما", "کمک", "قابلیت", "کارت چیه",
	"می تونی بکنی", "میتونی بکنی", "کاری می تونی", "کاری میتونی",
}

// isHelpQuery reports whether the text is a greeting or a capability question.
func isHelpQuery(text string) bool {
	if firstWholeWordIndex(text, greetingKeywords) != -1 {
		return true
	}
	return firstKeywordIndex(text, capabilityKeywords) != -1
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
