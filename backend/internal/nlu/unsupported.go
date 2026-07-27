package nlu

import "strings"

// DetectUnsupportedService returns a label for a travel service we don't offer
// when the text clearly asks for one, else "". This is for logging/analytics
// only — it deliberately does NOT influence detectIntent or the response, so it
// can never "steal" a supported request (there's a regression test). Whole-word
// matching keeps short words like "تور" from firing inside "دستور"/"موتور".
func DetectUnsupportedService(text string) string {
	n := Normalize(text)
	switch {
	case standaloneWordIndex(n, "تور") != -1:
		return "tour"
	case standaloneWordIndex(n, "ویلا") != -1,
		strings.Contains(n, "بوم گردی"),
		strings.Contains(n, "اقامتگاه بوم"):
		return "villa" // villa / ecolodge — not hotel search
	case standaloneWordIndex(n, "خارجی") != -1,
		strings.Contains(n, "بین المللی"):
		return "international" // international flight; we only do domestic
	}
	return ""
}
