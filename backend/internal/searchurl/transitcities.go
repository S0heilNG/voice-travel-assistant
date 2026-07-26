package searchurl

// Train and bus city slugs for 780.ir, captured from the live sitemaps and the
// pages' SSG route data. Two separate tables on purpose: the English spelling
// differs per service — اصفهان is "isfahan" for train but "esfahan" for bus,
// and اهواز is "ahvaz" vs "ahwaz". Sharing one table would build broken URLs.
//
// Keyed by the canonical Persian name (nlu.City.Name), not IATA: many rail/
// coach cities have no airport, so an IATA key can't represent them. Every key
// must be a real nlu city name (there's a test that checks this). A city is
// supported by a service iff it appears in that service's table.
//
// Only slugs confirmed from 780.ir's own data are here. Islands کیش/قشم have
// no rail/coach service and are absent from both.

var trainCities = map[string]string{
	"تهران":  "tehran",
	"مشهد":  "mashhad",
	"اصفهان": "isfahan", // bus uses "esfahan"
	"شیراز":  "shiraz",
	"تبریز":  "tabriz",
	"رشت":    "rasht",
	"یزد":    "yazd",
	"اهواز":  "ahvaz", // bus uses "ahwaz"
	"جلفا":   "jolfa",
	"زنجان":  "zanjan",
	"سمنان":  "semnan",
	"قزوین":  "qazvin",
	"کرج":    "karaj",
}

var busCities = map[string]string{
	"تهران":  "tehran",
	"مشهد":  "mashhad",
	"اصفهان": "esfahan", // train uses "isfahan"
	"شیراز":  "shiraz",
	"تبریز":  "tabriz",
	"رشت":    "rasht",
	"یزد":    "yazd",
	"اهواز":  "ahwaz", // train uses "ahvaz"
	"اردبیل": "ardabil",
	"بوشهر":  "bushehr",
	"ارومیه": "orumieh",
	"ساری":   "sari",
	"ایلام":  "ilam",
	"جهرم":   "jahrom",
	"رامسر":  "ramsar",
	"همدان":  "hamedan",
	"کاشان":  "kashan",
	"کرج":    "karaj",
}

// LookupTrainCity returns the 780.ir train slug for the given canonical name.
func LookupTrainCity(name string) (string, bool) {
	s, ok := trainCities[name]
	return s, ok
}

// LookupBusCity returns the 780.ir bus slug for the given canonical name.
func LookupBusCity(name string) (string, bool) {
	s, ok := busCities[name]
	return s, ok
}
