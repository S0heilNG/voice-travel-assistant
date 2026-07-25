package searchurl

// Train and bus city slugs for 780.ir, captured from the live sitemaps and the
// pages' SSG data. Two separate tables on purpose: the English spelling differs
// per service — اصفهان is "isfahan" for train but "esfahan" for bus, and اهواز
// is "ahvaz" vs "ahwaz". Guessing or sharing one table would build broken URLs.
//
// Only slugs we could confirm authoritatively are here. Anything that was a
// candidate/unverified (e.g. بندرعباس "bandar" vs "bndrabs", train ساری) is
// left out, so those cities read as "unsupported" — a broken redirect is worse
// than none (the hotel lesson). Islands کیش/قشم have no rail/coach service.
//
// Keyed by IATA to stay aligned with nlu.City without duplicating Persian
// spellings.
var trainCities = map[string]string{
	"THR": "tehran",
	"MHD": "mashhad",
	"IFN": "isfahan", // note: bus uses "esfahan"
	"SYZ": "shiraz",
	"TBZ": "tabriz",
	"RAS": "rasht",
	"AZD": "yazd",
	"AWZ": "ahvaz", // note: bus uses "ahwaz"
}

var busCities = map[string]string{
	"THR": "tehran",
	"MHD": "mashhad",
	"IFN": "esfahan", // note: train uses "isfahan"
	"SYZ": "shiraz",
	"TBZ": "tabriz",
	"RAS": "rasht",
	"AZD": "yazd",
	"AWZ": "ahwaz", // note: train uses "ahvaz"
	"ADU": "ardabil",
	"BUZ": "bushehr",
	"OMH": "orumieh",
}

// LookupTrainCity returns the 780.ir train slug for the given IATA code.
func LookupTrainCity(iata string) (string, bool) {
	s, ok := trainCities[iata]
	return s, ok
}

// LookupBusCity returns the 780.ir bus slug for the given IATA code.
func LookupBusCity(iata string) (string, bool) {
	s, ok := busCities[iata]
	return s, ok
}
