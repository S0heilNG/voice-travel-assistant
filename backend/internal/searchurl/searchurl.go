// Package searchurl builds 780.ir search result URLs from a resolved NLU
// parse result. It's kept separate from internal/nlu on purpose: nlu is
// about understanding Persian text (language in, structured intent out),
// while this package is about one specific downstream integration (780.ir's
// URL scheme). Keeping them apart means nlu doesn't need to know anything
// about 780.ir, and a future second consumer of nlu.ParseResult (or a
// second travel platform) wouldn't have to fight this package's assumptions.
package searchurl

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	jalaali "github.com/jalaali/go-jalaali"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"
)

// These mean we understood the city but 780.ir has no coverage for it on that
// service — a normal outcome to show the user, not a bug.
var (
	ErrHotelCityUnsupported  = fmt.Errorf("no 780.ir hotel support for this city")
	ErrTrainCityUnsupported  = fmt.Errorf("no 780.ir train support for this city")
	ErrBusCityUnsupported    = fmt.Errorf("no 780.ir bus support for this city")
	ErrFlightCityUnsupported = fmt.Errorf("no 780.ir flight support for this city")
	ErrTourDestUnsupported   = fmt.Errorf("no 780.ir tour for this destination")
)

// BuildTourSearchURL builds a 780.ir tour results URL. Tours are
// destination-only: no origin, no departure date (the results page filters by
// month, not day), so a destination alone is a complete request.
//
// The type comes from the destination table rather than from any separate
// domestic/international detection, because 780 is the one that decides which
// bucket a destination sits in — کربلا and نجف are INTERNATIONAL there.
//
// Returns ErrTourDestUnsupported for anything not in the table. That is a hard
// rule, not caution: 780's tour page does not 404 on an unknown slug, it hangs
// on a loading spinner forever, so a guessed slug would strand the user with no
// error at all.
func BuildTourSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentTourSearch {
		return "", fmt.Errorf("cannot build tour search URL: intent is %q, not %q", result.Intent, nlu.IntentTourSearch)
	}
	if result.Destination == nil {
		return "", fmt.Errorf("cannot build tour search URL: destination must be resolved")
	}
	dest, ok := LookupTourDestination(result.Destination.Name)
	if !ok {
		return "", ErrTourDestUnsupported
	}

	return fmt.Sprintf(
		"https://780.ir/tourism/tour/%s?type=%s&destinationName=%s",
		dest.Slug, dest.Type, url.QueryEscape(result.Destination.Name),
	), nil
}

// BuildFlightSearchURL builds a 780.ir flight search results URL from a
// parse result. Callers should check that result.Missing is empty and
// result.Intent is flight_search before calling this — it returns an error
// if intent, origin, destination, or date aren't all resolved.
func BuildFlightSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentFlightSearch {
		return "", fmt.Errorf("cannot build flight search URL: intent is %q, not %q", result.Intent, nlu.IntentFlightSearch)
	}
	if result.Origin == nil || result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build flight search URL: origin, destination, and date must all be resolved")
	}
	// A city with no confirmed airport has IATA "" — it isn't flight-able.
	if result.Origin.IATA == "" || result.Destination.IATA == "" {
		return "", ErrFlightCityUnsupported
	}

	return fmt.Sprintf(
		"https://780.ir/tourism/flights/%s-%s?adult=%d&child=0&infant=0&departureDate=%s&sort=lowPrice",
		result.Origin.IATA, result.Destination.IATA, result.Adults, result.Date.String(),
	), nil
}

// BuildInternationalFlightSearchURL builds a 780.ir international flight
// results URL.
//
// Two things differ from the domestic scheme and both were verified against
// 780's client bundle and a clean browser session:
//   - Dates are Jalali here too, despite the destination being abroad. The
//     bundle formats them with .calendar("jalali"), and the results page echoes
//     the Gregorian equivalent back, which confirmed it.
//   - The return-date parameter is spelled "returningDate", not "returnDate".
//     ("returnDate" is only 780's internal field name — using it in the URL
//     silently loses the return leg.)
//
// tripMode is 1 for one-way and 2 for round-trip; originType/destinationType
// are 0 because we always send airport codes, never city codes.
func BuildInternationalFlightSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentIntlFlightSearch {
		return "", fmt.Errorf("cannot build international flight search URL: intent is %q, not %q", result.Intent, nlu.IntentIntlFlightSearch)
	}
	if result.Origin == nil || result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build international flight search URL: origin, destination, and date must all be resolved")
	}
	if result.Origin.IATA == "" || result.Destination.IATA == "" {
		return "", ErrFlightCityUnsupported
	}

	tripMode := "1"
	returning := ""
	if result.ReturnDate != nil {
		tripMode = "2"
		returning = "&returningDate=" + result.ReturnDate.String()
	}

	return fmt.Sprintf(
		"https://780.ir/tourism/international/%s-%s?departureDate=%s%s"+
			"&cabinType=CABIN_TYPE_ECONOMY&adult=%d&child=0&infant=0"+
			"&originType=0&destinationType=0&tripMode=%s&sort=fast",
		result.Origin.IATA, result.Destination.IATA, result.Date.String(), returning,
		result.Adults, tripMode,
	), nil
}

// BuildTrainSearchURL builds a 780.ir train results URL. The path uses the
// train name-slugs (isfahan, ahvaz, ...); 780 fills in gender/wantCompartment
// defaults itself, so we only send the date and passenger counts. Returns
// ErrTrainCityUnsupported when either city has no train slug.
func BuildTrainSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentTrainSearch {
		return "", fmt.Errorf("cannot build train search URL: intent is %q, not %q", result.Intent, nlu.IntentTrainSearch)
	}
	if result.Origin == nil || result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build train search URL: origin, destination, and date must all be resolved")
	}
	o, ok1 := LookupTrainCity(result.Origin.Name)
	d, ok2 := LookupTrainCity(result.Destination.Name)
	if !ok1 || !ok2 {
		return "", ErrTrainCityUnsupported
	}
	return fmt.Sprintf(
		"https://780.ir/tourism/train/%s-%s?departureDate=%s&adult=%d&child=0&infant=0",
		o, d, result.Date.String(), result.Adults,
	), nil
}

// BuildBusSearchURL builds a 780.ir bus results URL. Bus needs the sort param
// (dropping it 404s the client route). Path uses the bus name-slugs (esfahan,
// ahwaz, ...). Returns ErrBusCityUnsupported when either city has no bus slug.
func BuildBusSearchURL(result nlu.ParseResult) (string, error) {
	if result.Intent != nlu.IntentBusSearch {
		return "", fmt.Errorf("cannot build bus search URL: intent is %q, not %q", result.Intent, nlu.IntentBusSearch)
	}
	if result.Origin == nil || result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build bus search URL: origin, destination, and date must all be resolved")
	}
	o, ok1 := LookupBusCity(result.Origin.Name)
	d, ok2 := LookupBusCity(result.Destination.Name)
	if !ok1 || !ok2 {
		return "", ErrBusCityUnsupported
	}
	return fmt.Sprintf(
		"https://780.ir/tourism/bus/%s-%s?departureDate=%s&sort=earliestTime",
		o, d, result.Date.String(),
	), nil
}

// AddJalaliDays shifts a Jalali date by n days. It converts to Gregorian,
// does the arithmetic on a real time.Time, and converts back, so month and
// year boundaries (and Esfand's 29/30 days in leap years) are handled by the
// calendar library rather than by hand-rolled month-length tables.
func AddJalaliDays(d nlu.JalaliDate, n int) (nlu.JalaliDate, error) {
	gy, gm, gd, err := jalaali.ToGregorian(d.Year, jalaali.Month(d.Month), d.Day)
	if err != nil {
		return nlu.JalaliDate{}, fmt.Errorf("converting %s to gregorian: %w", d, err)
	}
	// Noon keeps the arithmetic clear of any DST/midnight edge cases.
	shifted := time.Date(gy, gm, gd, 12, 0, 0, 0, time.UTC).AddDate(0, 0, n)
	jy, jm, jd, err := jalaali.ToJalaali(shifted.Year(), shifted.Month(), shifted.Day())
	if err != nil {
		return nlu.JalaliDate{}, fmt.Errorf("converting back to jalaali: %w", err)
	}
	return nlu.JalaliDate{Year: jy, Month: int(jm), Day: jd}, nil
}

// BuildHotelSearchURL builds a 780.ir hotel search results URL for the
// destination and check-in date in result, staying `nights` nights.
//
// Returns ErrHotelCityUnsupported when the destination is a city we
// understand but 780.ir has no hotel UUID for — callers should treat that as
// a message for the user, not a failure. Note requestId is deliberately
// omitted: results render fine without it.
func BuildHotelSearchURL(result nlu.ParseResult, nights int) (string, error) {
	if result.Intent != nlu.IntentHotelSearch {
		return "", fmt.Errorf("cannot build hotel search URL: intent is %q, not %q", result.Intent, nlu.IntentHotelSearch)
	}
	if result.Destination == nil || result.Date == nil {
		return "", fmt.Errorf("cannot build hotel search URL: destination and date must both be resolved")
	}
	if nights < 1 {
		nights = 1
	}

	hotelCity, ok := LookupHotelCity(result.Destination.Name)
	if !ok {
		return "", ErrHotelCityUnsupported
	}

	checkOut, err := AddJalaliDays(*result.Date, nights)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("checkInDate", result.Date.String())
	params.Set("checkOutDate", checkOut.String())
	params.Set("rooms", "A") // one room, one adult
	params.Set("destinationType", "city")
	params.Set("sort", "offer")
	params.Set("cityId", hotelCity.ID)
	params.Set("cityName", result.Destination.Name)
	params.Set("cityNameEng", hotelCity.NameEng)
	params.Set("readCache", "true")

	return fmt.Sprintf(
		"https://780.ir/tourism/hotel/search/%s?%s",
		strings.ToLower(hotelCity.NameEng), params.Encode(),
	), nil
}
