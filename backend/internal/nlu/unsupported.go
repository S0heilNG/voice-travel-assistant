package nlu

import "strings"

// DetectUnsupportedService returns a label for a travel service we don't offer
// when the text clearly asks for one, else "". This is for logging/analytics
// only — it deliberately does NOT influence detectIntent or the response, so it
// can never "steal" a supported request (there's a regression test).
//
// Only villas/ecolodges remain. "international" and "tour" were both reported
// here until each became a real service; leaving a label behind would inflate
// the "demand we can't serve" counts with demand we now serve. 780 has no
// standalone villa/ecolodge product at all (accommodation exists only inside a
// tour package), so this one has nowhere to graduate to.
func DetectUnsupportedService(text string) string {
	n := Normalize(text)
	switch {
	case standaloneWordIndex(n, "ویلا") != -1,
		strings.Contains(n, "بوم گردی"),
		strings.Contains(n, "اقامتگاه بوم"):
		return "villa" // villa / ecolodge — not hotel search
	}
	return ""
}
