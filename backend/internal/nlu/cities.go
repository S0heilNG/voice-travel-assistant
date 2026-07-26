package nlu

import (
	"sort"
	"strings"
)

// City is a resolved Iranian city. Name is the canonical Persian name — it is
// the key every downstream service (flight/train/bus/hotel) uses to look up
// its own identifier for the city, so those services no longer depend on the
// city having an airport. IATA is the domestic-flight code and is "" for
// cities 780.ir has no confirmed airport for (they simply aren't flight-able).
//
// TODO: IATA codes are from general knowledge, not validated against 780.ir's
// airport API (GET https://api.780.ir/domestic-flight-aggregator/v1/airports).
// New cities added for train/bus are intentionally left IATA:"" until their
// airports can be confirmed there — a broken flight URL is worse than none.
type City struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type cityEntry struct {
	city    City
	aliases []string
}

// cityEntries is the vocabulary. A city is here if any service supports it;
// which services actually cover it is decided by the per-service tables in
// internal/searchurl (keyed by Name), not by this list.
//
// Deliberately excluded because the name is a common Persian word and would
// misfire as a whole-word match: "خوی" (temperament) and "وان" (bathtub, and
// it's in Turkey anyway).
var cityEntries = []cityEntry{
	{City{"تهران", "THR"}, []string{"تهران", "تهرون"}},
	{City{"مشهد", "MHD"}, []string{"مشهد", "مشهد مقدس"}},
	{City{"شیراز", "SYZ"}, []string{"شیراز"}},
	{City{"اصفهان", "IFN"}, []string{"اصفهان", "اصفهون"}},
	{City{"تبریز", "TBZ"}, []string{"تبریز"}},
	{City{"کیش", "KIH"}, []string{"کیش", "جزیره کیش"}},
	{City{"اهواز", "AWZ"}, []string{"اهواز"}},
	{City{"بندرعباس", "BND"}, []string{"بندرعباس", "بندر عباس"}},
	{City{"رشت", "RAS"}, []string{"رشت"}},
	{City{"کرمان", "KER"}, []string{"کرمان"}},
	{City{"یزد", "AZD"}, []string{"یزد"}},
	{City{"قشم", "GSM"}, []string{"قشم", "جزیره قشم"}},
	{City{"ساری", "SRY"}, []string{"ساری"}},
	{City{"اردبیل", "ADU"}, []string{"اردبیل"}},
	{City{"زاهدان", "ZAH"}, []string{"زاهدان"}},
	{City{"کرمانشاه", "KSH"}, []string{"کرمانشاه"}},
	{City{"بوشهر", "BUZ"}, []string{"بوشهر"}},
	{City{"ارومیه", "OMH"}, []string{"ارومیه", "اورمیه"}},
	{City{"گرگان", "GBT"}, []string{"گرگان"}},
	{City{"بیرجند", "XBJ"}, []string{"بیرجند"}},
	// Added for train/bus coverage (Persian↔slug confirmed from 780.ir's SSG
	// route data). IATA left "" until the airport list confirms them.
	{City{"ایلام", ""}, []string{"ایلام"}},
	{City{"همدان", ""}, []string{"همدان"}},
	{City{"کاشان", ""}, []string{"کاشان"}},
	{City{"کرج", ""}, []string{"کرج"}},
	{City{"جهرم", ""}, []string{"جهرم"}},
	{City{"رامسر", ""}, []string{"رامسر"}},
	{City{"زنجان", ""}, []string{"زنجان"}},
	{City{"سمنان", ""}, []string{"سمنان"}},
	{City{"قزوین", ""}, []string{"قزوین"}},
	{City{"جلفا", ""}, []string{"جلفا"}},
}

var aliasToCity map[string]City

// aliasTokens is every alias pre-split into words, sorted longest-first so a
// greedy scan prefers "بندر عباس" over a hypothetical "بندر" and never lets a
// shorter name shadow a longer one.
type aliasTokenEntry struct {
	tokens []string
	city   City
}

var aliasTokens []aliasTokenEntry

func init() {
	aliasToCity = make(map[string]City, len(cityEntries)*2)
	for _, entry := range cityEntries {
		for _, alias := range entry.aliases {
			key := Normalize(alias)
			aliasToCity[key] = entry.city
			aliasTokens = append(aliasTokens, aliasTokenEntry{strings.Fields(key), entry.city})
		}
	}
	sort.Slice(aliasTokens, func(i, j int) bool {
		return len(aliasTokens[i].tokens) > len(aliasTokens[j].tokens)
	})
}

// LookupCity resolves a normalized city name/alias to a City. It expects an
// exact match (after Normalize) — e.g. "تهران" or "بندر عباس" — and returns
// nil if the name isn't recognized.
func LookupCity(name string) *City {
	name = Normalize(name)
	if c, ok := aliasToCity[name]; ok {
		city := c
		return &city
	}
	return nil
}

type cityMatch struct {
	city  City
	start int
}

// findCitiesInText scans normalized text for known cities using whole-word,
// longest-first, non-overlapping matching. Whole-word (not substring) is what
// stops "کرمان" from matching inside "کرمانشاه", "شوش" inside "شوشتر", or "بم"
// inside "بمب"; longest-first stops a multi-word name from being shadowed by a
// prefix. Returns matches in order of appearance.
func findCitiesInText(text string) []cityMatch {
	words := strings.Fields(text)

	// Byte offset of each word. text is Normalize'd (single-spaced, trimmed),
	// so a running "+len(word)+1 for the space" reproduces real offsets.
	offsets := make([]int, len(words))
	off := 0
	for i, w := range words {
		offsets[i] = off
		off += len(w) + len(" ")
	}

	var matches []cityMatch
	i := 0
	for i < len(words) {
		matched := false
		for _, ae := range aliasTokens { // longest-first
			n := len(ae.tokens)
			if i+n > len(words) {
				continue
			}
			ok := true
			for k := 0; k < n; k++ {
				if words[i+k] != ae.tokens[k] {
					ok = false
					break
				}
			}
			if ok {
				matches = append(matches, cityMatch{city: ae.city, start: offsets[i]})
				i += n // non-overlapping: skip the whole matched name
				matched = true
				break
			}
		}
		if !matched {
			i++
		}
	}
	return matches
}
