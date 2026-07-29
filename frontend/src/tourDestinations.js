// Tour destinations for 780.ir. Mirrors backend/internal/searchurl/
// tourdestinations.go — the frontend accumulates conversation context locally
// and redirects without a round trip, so it needs its own copy.
//
// Keyed by the canonical Persian destination name (slots.destination.name).
// The type is data, not something to infer: 780 decides which bucket a
// destination sits in, and it puts کربلا and نجف under INTERNATIONAL.
//
// ⚠️ Never guess a slug for a destination that isn't listed. An unknown slug
// does not 404 on 780 — the tour page spins on a loading state forever with no
// error — so an unlisted destination must produce the unsupported message
// instead of a link.
export const TOUR_DESTINATIONS = {
  // DOMESTIC
  مشهد: { slug: 'mashhad', type: 'DOMESTIC' },
  سرعین: { slug: 'sareyn', type: 'DOMESTIC' },
  ماسال: { slug: 'masal', type: 'DOMESTIC' },
  کیش: { slug: 'kish', type: 'DOMESTIC' },
  سوادکوه: { slug: 'savadkooh', type: 'DOMESTIC' },
  شیرگاه: { slug: 'shirgah', type: 'DOMESTIC' },
  رشت: { slug: 'rasht', type: 'DOMESTIC' },
  رامسر: { slug: 'ramsar', type: 'DOMESTIC' },
  قزوین: { slug: 'ghazvin', type: 'DOMESTIC' },
  ارومیه: { slug: 'orumie', type: 'DOMESTIC' },

  // INTERNATIONAL
  استانبول: { slug: 'istanbul', type: 'INTERNATIONAL' },
  کربلا: { slug: 'karbala', type: 'INTERNATIONAL' },
  آنتالیا: { slug: 'antalya', type: 'INTERNATIONAL' },
  تفلیس: { slug: 'tbilisi', type: 'INTERNATIONAL' },
  ایروان: { slug: 'yerevan', type: 'INTERNATIONAL' },
  باتومی: { slug: 'batumi', type: 'INTERNATIONAL' },
  نجف: { slug: 'najaf', type: 'INTERNATIONAL' },
  دبی: { slug: 'dubai', type: 'INTERNATIONAL' },
}

export function lookupTourDestination(name) {
  return TOUR_DESTINATIONS[name] ?? null
}

// Tours take neither an origin nor a departure date; the results page filters
// by month, not day. So a destination alone is a complete request.
export function buildTourSearchUrl(slots) {
  const dest = lookupTourDestination(slots.destination?.name)
  if (!dest) return null
  return (
    `https://780.ir/tourism/tour/${dest.slug}` +
    `?type=${dest.type}&destinationName=${encodeURIComponent(slots.destination.name)}`
  )
}
