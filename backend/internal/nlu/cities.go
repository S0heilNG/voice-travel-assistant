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

// cityEntries is the vocabulary. IATA codes come from 780.ir's own airport API
// (so the flight URLs are authoritative); "" means the city has no airport but
// is reachable by train/bus. Which services actually cover a city is decided by
// the per-service tables in internal/searchurl (keyed by Name), not by this
// list. Entries with IATA "" have no airport and are reachable only by
// train/bus (e.g. قم، بابل، آمل، قزوین، جلفا).
//
// Deliberately excluded because the name is a common Persian word that would
// misfire as a whole-word match, despite having an airport: "بم" (bass/low),
// "خوی" (temperament), plus "وان" (bathtub; Turkey). Also skipped the redundant
// second Bandar Abbas field (هوادریا) and an obscure military strip (بیشه‌کلا).
var cityEntries = []cityEntry{
	{City{"تهران", "THR"}, []string{"تهران", "تهرون"}},
	{City{"تبریز", "TBZ"}, []string{"تبریز"}},
	{City{"اصفهان", "IFN"}, []string{"اصفهان", "اصفهون"}},
	{City{"شیراز", "SYZ"}, []string{"شیراز"}},
	{City{"مشهد", "MHD"}, []string{"مشهد", "مشهد مقدس"}},
	{City{"اهواز", "AWZ"}, []string{"اهواز"}},
	{City{"اراک", "AJK"}, []string{"اراک"}},
	{City{"اردبیل", "ADU"}, []string{"اردبیل"}},
	{City{"بندرعباس", "BND"}, []string{"بندرعباس", "بندر عباس"}},
	{City{"رشت", "RAS"}, []string{"رشت"}},
	{City{"کرمان", "KER"}, []string{"کرمان"}},
	{City{"یزد", "AZD"}, []string{"یزد"}},
	{City{"ساری", "SRY"}, []string{"ساری"}},
	{City{"زاهدان", "ZAH"}, []string{"زاهدان"}},
	{City{"کرمانشاه", "KSH"}, []string{"کرمانشاه"}},
	{City{"بوشهر", "BUZ"}, []string{"بوشهر"}},
	{City{"ارومیه", "OMH"}, []string{"ارومیه", "اورمیه"}},
	{City{"گرگان", "GBT"}, []string{"گرگان"}},
	{City{"بیرجند", "XBJ"}, []string{"بیرجند"}},
	{City{"ایلام", "IIL"}, []string{"ایلام"}},
	{City{"همدان", "HDM"}, []string{"همدان"}},
	{City{"کاشان", "KKS"}, []string{"کاشان"}},
	{City{"کرج", "PYK"}, []string{"کرج"}},
	{City{"جهرم", "JAR"}, []string{"جهرم"}},
	{City{"رامسر", "RZR"}, []string{"رامسر"}},
	{City{"زنجان", "JWN"}, []string{"زنجان"}},
	{City{"سمنان", "SNX"}, []string{"سمنان"}},
	{City{"قزوین", ""}, []string{"قزوین"}},
	{City{"جلفا", ""}, []string{"جلفا"}},
	{City{"آبادان", "ABD"}, []string{"آبادان"}},
	{City{"سنندج", "SDG"}, []string{"سنندج"}},
	{City{"خرم آباد", "KHD"}, []string{"خرم آباد", "خرمآباد"}},
	{City{"یاسوج", "YES"}, []string{"یاسوج"}},
	{City{"سبزوار", "AFZ"}, []string{"سبزوار"}},
	{City{"مراغه", "ACP"}, []string{"مراغه"}},
	{City{"نوشهر", "NSH"}, []string{"نوشهر"}},
	{City{"چابهار", "ZBR"}, []string{"چابهار"}},
	{City{"دزفول", "DEF"}, []string{"دزفول"}},
	{City{"ماکو", "IMQ"}, []string{"ماکو"}},
	{City{"پارس آباد", "PFQ"}, []string{"پارس آباد"}},
	{City{"گناباد", "MDN"}, []string{"گناباد"}},
	{City{"سقز", "TQZ"}, []string{"سقز"}},
	{City{"طبس", "TCX"}, []string{"طبس"}},
	{City{"بجنورد", "BJB"}, []string{"بجنورد"}},
	{City{"رفسنجان", "RJN"}, []string{"رفسنجان"}},
	{City{"سیرجان", "SYJ"}, []string{"سیرجان"}},
	{City{"کنگان", "KNR"}, []string{"کنگان"}},
	{City{"لامرد", "LFM"}, []string{"لامرد"}},
	{City{"ماهشهر", "MRX"}, []string{"ماهشهر"}},
	{City{"ایرانشهر", "IHR"}, []string{"ایرانشهر"}},
	{City{"زابل", "ACZ"}, []string{"زابل"}},
	{City{"شهرکرد", "CQD"}, []string{"شهرکرد"}},
	{City{"لار", "LRR"}, []string{"لار"}},
	{City{"عسلویه", "PGU"}, []string{"عسلویه"}},
	{City{"بابلسر", "BBL"}, []string{"بابلسر"}},
	{City{"قم", ""}, []string{"قم"}},
	{City{"نیشابور", ""}, []string{"نیشابور"}},
	{City{"ساوه", ""}, []string{"ساوه"}},
	{City{"بابل", ""}, []string{"بابل"}},
	{City{"آمل", ""}, []string{"آمل"}},
	{City{"چالوس", ""}, []string{"چالوس"}},
	{City{"لاهیجان", ""}, []string{"لاهیجان"}},
	{City{"تنکابن", ""}, []string{"تنکابن"}},
	{City{"بروجرد", ""}, []string{"بروجرد"}},
	{City{"اندیمشک", ""}, []string{"اندیمشک"}},
	{City{"دورود", ""}, []string{"دورود"}},
	{City{"مرند", ""}, []string{"مرند"}},
	{City{"میانه", ""}, []string{"میانه"}},
	{City{"شاهرود", ""}, []string{"شاهرود"}},
	{City{"دامغان", ""}, []string{"دامغان"}},
	{City{"مهران", ""}, []string{"مهران"}},
	{City{"کیش", "KIH"}, []string{"کیش", "جزیره کیش"}},
	{City{"قشم", "GSM"}, []string{"قشم", "جزیره قشم"}},
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
