package nlu

import (
	"sort"
	"strings"
)

// TODO: the IATA codes below are from general knowledge and have NOT been
// validated against 780.ir's real airport data. Before this goes anywhere
// near production, cross-check every code against the actual API we found:
// GET https://api.780.ir/domestic-flight-aggregator/v1/airports?query=<name>
// and fix anything that's wrong.

// City is a resolved Iranian city with its domestic-flight IATA code.
type City struct {
	Name string
	IATA string
}

type cityEntry struct {
	city    City
	aliases []string
}

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
}

var aliasToCity map[string]City

func init() {
	aliasToCity = make(map[string]City, len(cityEntries)*2)
	for _, entry := range cityEntries {
		for _, alias := range entry.aliases {
			aliasToCity[alias] = entry.city
		}
	}
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

// findCitiesInText scans normalized free-form text for any known city
// alias and returns matches ordered by where they appear in the text.
func findCitiesInText(text string) []cityMatch {
	var matches []cityMatch
	for alias, city := range aliasToCity {
		if idx := strings.Index(text, alias); idx != -1 {
			matches = append(matches, cityMatch{city: city, start: idx})
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].start < matches[j].start })
	return matches
}
