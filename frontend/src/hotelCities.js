// 780.ir hotel integration data + the Jalali date arithmetic the hotel URL
// needs. Mirrors backend/internal/searchurl (hotelcities.go + searchurl.go);
// the frontend needs its own copy because it accumulates conversation context
// locally and redirects without a round trip.

// Only these 8 cities have 780.ir hotel coverage. Keyed by the canonical
// Persian city name (slots.destination.name), mirroring the backend table.
export const HOTEL_CITIES = {
  مشهد: { nameEng: 'Mashhad', id: 'ed48aafc-2d55-4a58-848c-4f0f7f0fd587' },
  کیش: { nameEng: 'Kish', id: '33efa9ce-ee1c-4e70-a612-abb2d5182c03' },
  اصفهان: { nameEng: 'Isfahan', id: '84e89d24-f31f-40f2-aac2-bd03a9dc8c3c' },
  شیراز: { nameEng: 'Shiraz', id: '15c8c243-d23f-4a3d-8338-075fc5f7cfbb' },
  تهران: { nameEng: 'Tehran', id: 'eb5f0fdb-c170-49a4-bab0-3e6ca0b12e03' },
  تبریز: { nameEng: 'Tabriz', id: '7d4d062d-a035-4278-b6f6-3e6b2c2ff925' },
  یزد: { nameEng: 'Yazd', id: '556b7d98-ca7f-44d8-9a1f-ed592d539338' },
  قشم: { nameEng: 'Qeshm', id: '299da048-9776-4ce9-b764-64a3de1497c9' },
}

// --- Jalali date arithmetic -------------------------------------------------
// Port of the standard jalaali algorithm (jalaali-js / jalaali-go share it).
// We go through Julian Day Numbers rather than month-length tables so month
// ends, year ends and Esfand's 29-vs-30 days in leap years all fall out of the
// calendar maths instead of needing special cases. Cross-checked against the
// Go backend's go-jalaali output for every day of 1395–1425.

const BREAKS = [
  -61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394,
  2456, 3178,
]

// These must truncate toward zero, not floor. The algorithm feeds them
// negative values (e.g. div(gm - 8, 6) for Jan–Jul), where floor would be off
// by one and shift the result by a whole year.
const div = (a, b) => Math.trunc(a / b)
const mod = (a, b) => a - Math.trunc(a / b) * b

// Leap-year offset data for a Jalali year.
function jalCal(jy) {
  const bl = BREAKS.length
  const gy = jy + 621
  let jump = 0
  let jp = BREAKS[0]
  let leapJ = -14

  if (jy < jp || jy >= BREAKS[bl - 1]) throw new Error(`invalid jalali year ${jy}`)

  for (let i = 1; i < bl; i += 1) {
    const jm = BREAKS[i]
    jump = jm - jp
    if (jy < jm) break
    leapJ += div(jump, 33) * 8 + div(mod(jump, 33), 4)
    jp = jm
  }
  let n = jy - jp

  leapJ += div(n, 33) * 8 + div(mod(n, 33) + 3, 4)
  if (mod(jump, 33) === 4 && jump - n === 4) leapJ += 1

  const leapG = div(gy, 4) - div((div(gy, 100) + 1) * 3, 4) - 150
  const march = 20 + leapJ - leapG

  if (jump - n < 6) n = n - jump + div(jump + 4, 33) * 33
  let leap = mod(mod(n + 1, 33) - 1, 4)
  if (leap === -1) leap = 4

  return { leap, gy, march }
}

// Gregorian y/m/d → Julian Day Number.
function g2d(gy, gm, gd) {
  let d =
    div((gy + div(gm - 8, 6) + 100100) * 1461, 4) +
    div(153 * mod(gm + 9, 12) + 2, 5) +
    gd -
    34840408
  d = d - div(div(gy + 100100 + div(gm - 8, 6), 100) * 3, 4) + 752
  return d
}

// Julian Day Number → Gregorian y/m/d.
function d2g(jdn) {
  let j = 4 * jdn + 139361631
  j += div(div(4 * jdn + 183187720, 146097) * 3, 4) * 4 - 3908
  const i = div(mod(j, 1461), 4) * 5 + 308
  const gd = div(mod(i, 153), 5) + 1
  const gm = mod(div(i, 153), 12) + 1
  const gy = div(j, 1461) - 100100 + div(8 - gm, 6)
  return { gy, gm, gd }
}

// Jalali y/m/d → Julian Day Number.
function j2d(jy, jm, jd) {
  const r = jalCal(jy)
  return g2d(r.gy, 3, r.march) + (jm - 1) * 31 - div(jm, 7) * (jm - 7) + jd - 1
}

// Julian Day Number → Jalali y/m/d.
function d2j(jdn) {
  const gy = d2g(jdn).gy
  let jy = gy - 621
  const r = jalCal(jy)
  const jdn1f = g2d(gy, 3, r.march)

  let k = jdn - jdn1f
  if (k >= 0) {
    if (k <= 185) {
      const jm = 1 + div(k, 31)
      const jd = mod(k, 31) + 1
      return { jy, jm, jd }
    }
    k -= 186
  } else {
    // Previous Jalali year. Note this uses r.leap from the year we started
    // with, not a recomputed one — getting that wrong shifts every date in
    // the last months of a leap year by a day.
    jy -= 1
    k += 179
    if (r.leap === 1) k += 1
  }
  const jm = 7 + div(k, 30)
  const jd = mod(k, 30) + 1
  return { jy, jm, jd }
}

const pad = (n) => String(n).padStart(2, '0')

/**
 * Shifts a Jalali "YYYY-MM-DD" string by n days, returning the same format.
 * e.g. addJalaliDays('1405-04-31', 1) === '1405-05-01'
 */
export function addJalaliDays(date, n) {
  const [jy, jm, jd] = date.split('-').map(Number)
  const { jy: y, jm: m, jd: d } = d2j(j2d(jy, jm, jd) + n)
  return `${y}-${pad(m)}-${pad(d)}`
}

/** 780.ir hotel identity for a canonical Persian city name, or null. */
export function lookupHotelCity(name) {
  return HOTEL_CITIES[name] || null
}

/**
 * Builds the 780.ir hotel URL for the destination. Returns null when the city
 * has no hotel coverage or the slots aren't complete.
 *
 * TEMPORARY: 780's hotel *results* page
 * (/tourism/hotel/search/{city}?checkInDate=...&cityId=...) ignores its URL
 * query params in a clean browser session — the destination comes up
 * `undefined` and the dates reset to defaults (verified in Incognito; failed
 * with requestId, without it, and with readCache=false). Our earlier "working"
 * tests were polluted by session cache. So instead of dropping the user on a
 * broken results page, we send them to the *landing* page, which correctly
 * resolves the city and lists its hotels; the confirm card tells them to pick
 * the (already-known) dates there. The nights/date logic is kept intact and
 * unused here — it comes back the moment 780 gives us the official hotel
 * deep-link format. See CLAUDE.md "یکپارچه‌سازی با ۷۸۰".
 */
export function buildHotelSearchUrl(slots) {
  if (!slots || !slots.destination || !slots.date) return null
  const hotelCity = lookupHotelCity(slots.destination.name)
  if (!hotelCity) return null
  return `https://780.ir/tourism/hotel/${hotelCity.nameEng.toLowerCase()}`
}
