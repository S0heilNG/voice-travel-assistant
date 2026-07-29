package nlu

import (
	"sort"
	"strings"
)

// Tour destinations have their own vocabulary because they only partly overlap
// with the city one: 780 sells tours to سرعین، ماسال، سوادکوه and شیرگاه, which
// no flight/train/bus/hotel service covers, while most cities in cities.go have
// no tour at all.
//
// They are kept OUT of the shared city vocabulary on purpose. Adding them there
// would make "سرعین" on its own parse as a flight search (the no-keyword
// fallback treats any known city as one) for a town with no airport — a
// guaranteed dead end. Scoping them to the tour intent avoids that entirely.
//
// The canonical names here must match the keys of searchurl.tourDestinations;
// a test enforces it.
var tourDestEntries = []cityEntry{
	{City{"مشهد", "MHD"}, []string{"مشهد", "مشهد مقدس"}},
	{City{"سرعین", ""}, []string{"سرعین"}},
	{City{"ماسال", ""}, []string{"ماسال"}},
	{City{"کیش", "KIH"}, []string{"کیش", "جزیره کیش"}},
	{City{"سوادکوه", ""}, []string{"سوادکوه"}},
	{City{"شیرگاه", ""}, []string{"شیرگاه"}},
	{City{"رشت", "RAS"}, []string{"رشت"}},
	{City{"رامسر", ""}, []string{"رامسر"}},
	{City{"قزوین", ""}, []string{"قزوین"}},
	{City{"ارومیه", "OMH"}, []string{"ارومیه", "اورمیه"}},

	{City{"استانبول", "IST"}, []string{"استانبول", "اسلامبول"}},
	{City{"کربلا", ""}, []string{"کربلا"}},
	{City{"آنتالیا", "AYT"}, []string{"آنتالیا", "انتالیا"}},
	{City{"تفلیس", "TBS"}, []string{"تفلیس", "تبلیسی"}},
	{City{"ایروان", "EVN"}, []string{"ایروان"}},
	{City{"باتومی", ""}, []string{"باتومی", "باتوم"}},
	{City{"نجف", ""}, []string{"نجف"}},
	{City{"دبی", "DXB"}, []string{"دبی"}},
}

var tourAliasTokens []aliasTokenEntry

func init() {
	for _, entry := range tourDestEntries {
		for _, alias := range entry.aliases {
			key := Normalize(alias)
			tourAliasTokens = append(tourAliasTokens, aliasTokenEntry{strings.Fields(key), entry.city})
		}
	}
	sort.Slice(tourAliasTokens, func(i, j int) bool {
		return len(tourAliasTokens[i].tokens) > len(tourAliasTokens[j].tokens)
	})
}

// IsTourDestinationName reports whether a canonical name is in the tour
// vocabulary. It exists so searchurl can assert that its slug table and this
// vocabulary describe the same set of destinations.
func IsTourDestinationName(name string) bool {
	for _, e := range tourDestEntries {
		if e.city.Name == name {
			return true
		}
	}
	return false
}

// extractTourDestination resolves the destination of a tour request.
//
// A bookable destination wins. Failing that it falls back to the general city
// vocabulary, so "تور تهران" still resolves to a city — one that has no tour,
// which the URL builder then reports as unsupported. Without that fallback the
// destination would come back nil and the assistant would ask "کدوم مقصد؟"
// again after the user had already answered, looping forever.
func extractTourDestination(text string) *City {
	if m := findInVocabulary(text, tourAliasTokens); len(m) > 0 {
		c := m[0].city
		return &c
	}
	if m := findCitiesInText(text); len(m) > 0 {
		c := m[0].city
		return &c
	}
	return nil
}
