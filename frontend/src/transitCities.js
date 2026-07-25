// Train and bus city slugs for 780.ir. Mirrors backend/internal/searchurl/
// transitcities.go — the frontend accumulates conversation context locally and
// redirects without a round trip, so it needs its own copy.
//
// Two separate tables because the English spelling differs per service:
// اصفهان = isfahan (train) vs esfahan (bus); اهواز = ahvaz (train) vs ahwaz
// (bus). Only authoritatively-confirmed slugs are here; unconfirmed cities are
// treated as unsupported (a broken redirect is worse than none). Keyed by IATA
// to stay aligned with the NLU city table. Islands کیش/قشم have no rail/coach.

export const TRAIN_CITIES = {
  THR: 'tehran',
  MHD: 'mashhad',
  IFN: 'isfahan', // bus uses "esfahan"
  SYZ: 'shiraz',
  TBZ: 'tabriz',
  RAS: 'rasht',
  AZD: 'yazd',
  AWZ: 'ahvaz', // bus uses "ahwaz"
}

export const BUS_CITIES = {
  THR: 'tehran',
  MHD: 'mashhad',
  IFN: 'esfahan', // train uses "isfahan"
  SYZ: 'shiraz',
  TBZ: 'tabriz',
  RAS: 'rasht',
  AZD: 'yazd',
  AWZ: 'ahwaz', // train uses "ahvaz"
  ADU: 'ardabil',
  BUZ: 'bushehr',
  OMH: 'orumieh',
}

export function lookupTrainCity(iata) {
  return TRAIN_CITIES[iata] || null
}

export function lookupBusCity(iata) {
  return BUS_CITIES[iata] || null
}

// 780 fills gender/wantCompartment defaults itself, so we only send date +
// passengers. Returns null if incomplete or either city has no train slug.
export function buildTrainSearchUrl(slots) {
  if (!slots || !slots.origin || !slots.destination || !slots.date) return null
  const o = lookupTrainCity(slots.origin.iata)
  const d = lookupTrainCity(slots.destination.iata)
  if (!o || !d) return null
  return (
    `https://780.ir/tourism/train/${o}-${d}` +
    `?departureDate=${slots.date}&adult=${slots.adults ?? 1}&child=0&infant=0`
  )
}

// Bus needs the sort param (dropping it 404s the client route).
export function buildBusSearchUrl(slots) {
  if (!slots || !slots.origin || !slots.destination || !slots.date) return null
  const o = lookupBusCity(slots.origin.iata)
  const d = lookupBusCity(slots.destination.iata)
  if (!o || !d) return null
  return `https://780.ir/tourism/bus/${o}-${d}?departureDate=${slots.date}&sort=earliestTime`
}
