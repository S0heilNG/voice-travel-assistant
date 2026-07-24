package nlu

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// spelledNumbers maps written Persian numbers (one through ten) to their
// value, including a couple of colloquial variants ("یه" for one, "شیش" for
// six). Enough for a realistic number of hotel nights.
var spelledNumbers = map[string]int{
	"یک": 1, "یه": 1, "دو": 2, "سه": 3, "چهار": 4, "پنج": 5,
	"شش": 6, "شیش": 6, "هفت": 7, "هشت": 8, "نه": 9, "ده": 10,
}

// nightsDigitRe matches a digit count of nights, e.g. "3 شب" or "2 شبه"
// (digits are already Latin at this point thanks to Normalize).
var nightsDigitRe = regexp.MustCompile(`(\d{1,3})\s*شب`)

// parseNights extracts a nights count from normalized text. It understands
// both digit forms ("3 شب", "2 شبه") and spelled forms ("سه شب", "یه شب").
// Returns 0 when no nights count is present.
func parseNights(text string) int {
	if m := nightsDigitRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n >= 1 && n <= 90 {
			return n
		}
	}

	words := strings.Split(text, " ")
	for i := 1; i < len(words); i++ {
		// "شب" (night) or the colloquial "شبه" ("...it's N nights").
		if words[i] == "شب" || words[i] == "شبه" {
			if n, ok := spelledNumbers[words[i-1]]; ok {
				return n
			}
		}
	}
	return 0
}

// parseDateRange looks for a "<date> تا <date>" / "<date> الی <date>" range and
// returns both endpoints. It splits on the first "تا"/"الی" whose two sides
// each contain a parseable date; requiring both sides to be dates keeps the
// very common word "تا" from producing false ranges. Returns nil,nil if no
// such range exists.
func parseDateRange(text string, now time.Time) (checkIn, checkOut *JalaliDate) {
	words := strings.Split(text, " ")
	for i, w := range words {
		if w != "تا" && w != "الی" {
			continue
		}
		before := strings.Join(words[:i], " ")
		after := strings.Join(words[i+1:], " ")
		ci := ParseDate(before, now)
		co := ParseDate(after, now)
		if ci != nil && co != nil {
			return ci, co
		}
	}
	return nil, nil
}

// nightsBetween returns the number of nights from a (check-in) to b
// (check-out): the count of calendar days between them. Both convert to
// Gregorian midnight, so the difference is an exact number of days. A
// non-positive result means b is not after a.
func nightsBetween(a, b JalaliDate) int {
	ga, err := a.toGregorian()
	if err != nil {
		return 0
	}
	gb, err := b.toGregorian()
	if err != nil {
		return 0
	}
	return int(gb.Sub(ga).Hours()/24 + 0.5)
}

// parseHotelStay resolves a hotel stay's check-in date and number of nights
// from normalized text. A date range ("از ۲۰ مرداد تا ۲۵ مرداد") yields the
// check-in plus the nights spanned; otherwise it's a single check-in date
// plus an explicit nights count ("۳ شب") if given. Either value can come back
// unset (nil date / 0 nights) for the caller to treat as a missing field.
func parseHotelStay(text string, now time.Time) (checkIn *JalaliDate, nights int) {
	if ci, co := parseDateRange(text, now); ci != nil && co != nil {
		n := nightsBetween(*ci, *co)
		if n < 1 {
			// Check-out looks to be on/before check-in — most likely the
			// range crosses the new year and the check-out date resolved to
			// the current year. Try the next year.
			bumped := *co
			bumped.Year++
			n = nightsBetween(*ci, bumped)
		}
		if n >= 1 {
			return ci, n
		}
	}

	return ParseDate(text, now), parseNights(text)
}
