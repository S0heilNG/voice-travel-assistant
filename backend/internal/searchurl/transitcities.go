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
	"تهران":   "tehran",
	"تبریز":   "tabriz",
	"اصفهان":  "isfahan",
	"شیراز":   "shiraz",
	"مشهد":    "mashhad",
	"اهواز":   "ahvaz",
	"اراک":    "arak",
	"رشت":     "rasht",
	"کرمان":   "kerman",
	"یزد":     "yazd",
	"ساری":    "sari",
	"زاهدان":  "zahedan",
	"ارومیه":  "urmia",
	"گرگان":   "gorgan",
	"همدان":   "hamedan",
	"کاشان":   "kashan",
	"کرج":     "karaj",
	"زنجان":   "zanjan",
	"سمنان":   "semnan",
	"قزوین":   "qazvin",
	"جلفا":    "jolfa",
	"سبزوار":  "sabzevar",
	"مراغه":   "marageh",
	"طبس":     "tabas",
	"سیرجان":  "sirjan",
	"ماهشهر":  "mahshahr",
	"شهرکرد":  "shahrekord",
	"قم":      "qom",
	"نیشابور": "neyshabur",
	"ساوه":    "saveh",
	"اندیمشک": "andimeshk",
	"دورود":   "dorud",
	"میانه":   "mianeh",
	"شاهرود":  "shahrud",
	"دامغان":  "damghan",
	"مهران":   "mehran",
}

var busCities = map[string]string{
	"تهران":    "tehran",
	"تبریز":    "tabriz",
	"اصفهان":   "esfahan",
	"شیراز":    "shiraz",
	"مشهد":     "mashhad",
	"اهواز":    "ahwaz",
	"اراک":     "arak",
	"اردبیل":   "ardabil",
	"رشت":      "rasht",
	"کرمان":    "kerman",
	"یزد":      "yazd",
	"ساری":     "sari",
	"زاهدان":   "zahedan",
	"کرمانشاه": "kermanshah",
	"بوشهر":    "bushehr",
	"ارومیه":   "orumieh",
	"گرگان":    "gorgan",
	"بیرجند":   "birjand",
	"ایلام":    "ilam",
	"همدان":    "hamedan",
	"کاشان":    "kashan",
	"کرج":      "karaj",
	"جهرم":     "jahrom",
	"رامسر":    "ramsar",
	"زنجان":    "zanjan",
	"سمنان":    "semnan",
	"قزوین":    "qazvin",
	"جلفا":     "julfa",
	"آبادان":   "abadan",
	"سنندج":    "sanandaj",
	"خرم آباد": "khorramabad",
	"یاسوج":    "yasuj",
	"سبزوار":   "sabzevar",
	"مراغه":    "maragheh",
	"نوشهر":    "nowshahr",
	"چابهار":   "chabahar",
	"دزفول":    "dezful",
	"ماکو":     "mako",
	"گناباد":   "gonabad",
	"سقز":      "saqez",
	"طبس":      "tabas",
	"بجنورد":   "bojnourd",
	"رفسنجان":  "rafsanjan",
	"سیرجان":   "sirjan",
	"کنگان":    "kangan",
	"لامرد":    "lamerd",
	"ماهشهر":   "mahshahr",
	"ایرانشهر": "iranshahr",
	"زابل":     "zabol",
	"عسلویه":   "assaluyeh",
	"بابلسر":   "babolsar",
	"قم":       "qom",
	"بابل":     "babol",
	"آمل":      "amol",
	"چالوس":    "chalous",
	"لاهیجان":  "lahijan",
	"تنکابن":   "tonekabon",
	"بروجرد":   "boroujerd",
	"اندیمشک":  "andimeshk",
	"مرند":     "marand",
	"میانه":    "miyane",
	"دامغان":   "damghan",
	"مهران":    "mehran",
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
