// Train and bus city slugs for 780.ir. Mirrors backend/internal/searchurl/
// transitcities.go — the frontend accumulates conversation context locally and
// redirects without a round trip, so it needs its own copy.
//
// Two separate tables because the English spelling differs per service:
// اصفهان = isfahan (train) vs esfahan (bus); اهواز = ahvaz (train) vs ahwaz
// (bus). Keyed by the canonical Persian city name (slots.*.name), not IATA, so
// airport-less rail/coach cities work too. Only confirmed slugs are here;
// anything else is treated as unsupported. Islands کیش/قشم have no rail/coach.

export const TRAIN_CITIES = {
  "تهران": "tehran",
  "تبریز": "tabriz",
  "اصفهان": "isfahan",
  "شیراز": "shiraz",
  "مشهد": "mashhad",
  "اهواز": "ahvaz",
  "اراک": "arak",
  "رشت": "rasht",
  "کرمان": "kerman",
  "یزد": "yazd",
  "ساری": "sari",
  "زاهدان": "zahedan",
  "ارومیه": "urmia",
  "گرگان": "gorgan",
  "همدان": "hamedan",
  "کاشان": "kashan",
  "کرج": "karaj",
  "زنجان": "zanjan",
  "سمنان": "semnan",
  "قزوین": "qazvin",
  "جلفا": "jolfa",
  "سبزوار": "sabzevar",
  "مراغه": "marageh",
  "طبس": "tabas",
  "سیرجان": "sirjan",
  "ماهشهر": "mahshahr",
  "شهرکرد": "shahrekord",
  "قم": "qom",
  "نیشابور": "neyshabur",
  "ساوه": "saveh",
  "اندیمشک": "andimeshk",
  "دورود": "dorud",
  "میانه": "mianeh",
  "شاهرود": "shahrud",
  "دامغان": "damghan",
  "مهران": "mehran",
}

export const BUS_CITIES = {
  "تهران": "tehran",
  "تبریز": "tabriz",
  "اصفهان": "esfahan",
  "شیراز": "shiraz",
  "مشهد": "mashhad",
  "اهواز": "ahwaz",
  "اراک": "arak",
  "اردبیل": "ardabil",
  "رشت": "rasht",
  "کرمان": "kerman",
  "یزد": "yazd",
  "ساری": "sari",
  "زاهدان": "zahedan",
  "کرمانشاه": "kermanshah",
  "بوشهر": "bushehr",
  "ارومیه": "orumieh",
  "گرگان": "gorgan",
  "بیرجند": "birjand",
  "ایلام": "ilam",
  "همدان": "hamedan",
  "کاشان": "kashan",
  "کرج": "karaj",
  "جهرم": "jahrom",
  "رامسر": "ramsar",
  "زنجان": "zanjan",
  "سمنان": "semnan",
  "قزوین": "qazvin",
  "جلفا": "julfa",
  "آبادان": "abadan",
  "سنندج": "sanandaj",
  "خرم آباد": "khorramabad",
  "یاسوج": "yasuj",
  "سبزوار": "sabzevar",
  "مراغه": "maragheh",
  "نوشهر": "nowshahr",
  "چابهار": "chabahar",
  "دزفول": "dezful",
  "ماکو": "mako",
  "گناباد": "gonabad",
  "سقز": "saqez",
  "طبس": "tabas",
  "بجنورد": "bojnourd",
  "رفسنجان": "rafsanjan",
  "سیرجان": "sirjan",
  "کنگان": "kangan",
  "لامرد": "lamerd",
  "ماهشهر": "mahshahr",
  "ایرانشهر": "iranshahr",
  "زابل": "zabol",
  "عسلویه": "assaluyeh",
  "بابلسر": "babolsar",
  "قم": "qom",
  "بابل": "babol",
  "آمل": "amol",
  "چالوس": "chalous",
  "لاهیجان": "lahijan",
  "تنکابن": "tonekabon",
  "بروجرد": "boroujerd",
  "اندیمشک": "andimeshk",
  "مرند": "marand",
  "میانه": "miyane",
  "دامغان": "damghan",
  "مهران": "mehran",
}

export function lookupTrainCity(name) {
  return TRAIN_CITIES[name] || null
}

export function lookupBusCity(name) {
  return BUS_CITIES[name] || null
}

// 780 fills gender/wantCompartment defaults itself, so we only send date +
// passengers. Returns null if incomplete or either city has no train slug.
export function buildTrainSearchUrl(slots) {
  if (!slots || !slots.origin || !slots.destination || !slots.date) return null
  const o = lookupTrainCity(slots.origin.name)
  const d = lookupTrainCity(slots.destination.name)
  if (!o || !d) return null
  return (
    `https://780.ir/tourism/train/${o}-${d}` +
    `?departureDate=${slots.date}&adult=${slots.adults ?? 1}&child=0&infant=0`
  )
}

// Bus needs the sort param (dropping it 404s the client route).
export function buildBusSearchUrl(slots) {
  if (!slots || !slots.origin || !slots.destination || !slots.date) return null
  const o = lookupBusCity(slots.origin.name)
  const d = lookupBusCity(slots.destination.name)
  if (!o || !d) return null
  return `https://780.ir/tourism/bus/${o}-${d}?departureDate=${slots.date}&sort=earliestTime`
}
