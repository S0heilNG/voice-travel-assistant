package searchurl

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"
)

func TestAddJalaliDays(t *testing.T) {
	tests := []struct {
		name string
		in   nlu.JalaliDate
		days int
		want nlu.JalaliDate
	}{
		{"mid-month", nlu.JalaliDate{Year: 1405, Month: 4, Day: 15}, 1, nlu.JalaliDate{Year: 1405, Month: 4, Day: 16}},
		// Months 1-6 have 31 days, so Tir 31 is the last day of the month.
		{"end of 31-day month", nlu.JalaliDate{Year: 1405, Month: 4, Day: 31}, 1, nlu.JalaliDate{Year: 1405, Month: 5, Day: 1}},
		// Months 7-11 have 30 days.
		{"end of 30-day month", nlu.JalaliDate{Year: 1405, Month: 8, Day: 30}, 1, nlu.JalaliDate{Year: 1405, Month: 9, Day: 1}},
		{"end of Bahman", nlu.JalaliDate{Year: 1405, Month: 11, Day: 30}, 1, nlu.JalaliDate{Year: 1405, Month: 12, Day: 1}},
		// 1403 is a leap year: Esfand has 30 days.
		{"leap year Esfand 29 → 30", nlu.JalaliDate{Year: 1403, Month: 12, Day: 29}, 1, nlu.JalaliDate{Year: 1403, Month: 12, Day: 30}},
		{"leap year end → new year", nlu.JalaliDate{Year: 1403, Month: 12, Day: 30}, 1, nlu.JalaliDate{Year: 1404, Month: 1, Day: 1}},
		// 1404 is not a leap year: Esfand has 29 days.
		{"non-leap year end → new year", nlu.JalaliDate{Year: 1404, Month: 12, Day: 29}, 1, nlu.JalaliDate{Year: 1405, Month: 1, Day: 1}},
		{"multi-night across month", nlu.JalaliDate{Year: 1405, Month: 4, Day: 30}, 3, nlu.JalaliDate{Year: 1405, Month: 5, Day: 2}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AddJalaliDays(tc.in, tc.days)
			if err != nil {
				t.Fatalf("AddJalaliDays(%s, %d) returned error: %v", tc.in, tc.days, err)
			}
			if got != tc.want {
				t.Errorf("AddJalaliDays(%s, %d) = %s, want %s", tc.in, tc.days, got, tc.want)
			}
		})
	}
}

func hotelResult(name string, d nlu.JalaliDate) nlu.ParseResult {
	return nlu.ParseResult{
		Intent:      nlu.IntentHotelSearch,
		Destination: &nlu.City{Name: name},
		Date:        &d,
		Adults:      1,
	}
}

func TestBuildHotelSearchURL(t *testing.T) {
	got, err := BuildHotelSearchURL(hotelResult("مشهد", nlu.JalaliDate{Year: 1405, Month: 4, Day: 31}), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(got, "https://780.ir/tourism/hotel/search/mashhad?") {
		t.Errorf("path should use the lowercased English name, got %s", got)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("built an unparseable URL: %v", err)
	}
	q := u.Query()
	want := map[string]string{
		"checkInDate":     "1405-04-31",
		"checkOutDate":    "1405-05-01", // month boundary, not 1405-04-32
		"rooms":           "A",
		"destinationType": "city",
		"sort":            "offer",
		"cityId":          "ed48aafc-2d55-4a58-848c-4f0f7f0fd587",
		"cityName":        "مشهد",
		"cityNameEng":     "Mashhad",
		"readCache":       "true",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("param %s = %q, want %q", k, q.Get(k), v)
		}
	}
	if q.Has("requestId") {
		t.Errorf("requestId should be omitted, got %q", q.Get("requestId"))
	}
}

func TestBuildHotelSearchURLUnsupportedCity(t *testing.T) {
	// رشت is a known flight city but has no 780.ir hotel UUID.
	_, err := BuildHotelSearchURL(hotelResult("رشت", nlu.JalaliDate{Year: 1405, Month: 4, Day: 31}), 1)
	if !errors.Is(err, ErrHotelCityUnsupported) {
		t.Fatalf("want ErrHotelCityUnsupported, got %v", err)
	}
}

func TestBuildHotelSearchURLRejectsIncomplete(t *testing.T) {
	flight := nlu.ParseResult{Intent: nlu.IntentFlightSearch}
	if _, err := BuildHotelSearchURL(flight, 1); err == nil {
		t.Error("expected an error for a flight intent")
	}

	noDate := nlu.ParseResult{
		Intent:      nlu.IntentHotelSearch,
		Destination: &nlu.City{Name: "مشهد", IATA: "MHD"},
	}
	if _, err := BuildHotelSearchURL(noDate, 1); err == nil {
		t.Error("expected an error when the date is missing")
	}
}

// Every service-table key must be a real canonical nlu city name, or the
// lookup can never fire. Catches typos and normalization drift.
func TestServiceTableKeysResolveInNLU(t *testing.T) {
	for tableName, table := range map[string]map[string]string{
		"train": trainCities,
		"bus":   busCities,
	} {
		for name := range table {
			city := nlu.LookupCity(name)
			if city == nil {
				t.Errorf("%s table key %q is not a known nlu city", tableName, name)
				continue
			}
			if city.Name != name {
				t.Errorf("%s table key %q is not the canonical name (nlu canonical is %q)", tableName, name, city.Name)
			}
		}
	}
	for name := range hotelCities {
		city := nlu.LookupCity(name)
		if city == nil || city.Name != name {
			t.Errorf("hotel table key %q is not a canonical nlu city name", name)
		}
	}
}

func routeResult(intent nlu.Intent, origin, dest string, d nlu.JalaliDate) nlu.ParseResult {
	return nlu.ParseResult{
		Intent:      intent,
		Origin:      &nlu.City{Name: origin},
		Destination: &nlu.City{Name: dest},
		Date:        &d,
		Adults:      1,
	}
}

func TestBuildTrainSearchURL(t *testing.T) {
	d := nlu.JalaliDate{Year: 1405, Month: 5, Day: 20}
	got, err := BuildTrainSearchURL(routeResult(nlu.IntentTrainSearch, "تهران", "اصفهان", d))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// اصفهان must be "isfahan" for TRAIN.
	want := "https://780.ir/tourism/train/tehran-isfahan?departureDate=1405-05-20&adult=1&child=0&infant=0"
	if got != want {
		t.Errorf("train URL =\n  %s\nwant\n  %s", got, want)
	}
}

func TestBuildBusSearchURL(t *testing.T) {
	d := nlu.JalaliDate{Year: 1405, Month: 5, Day: 21}
	got, err := BuildBusSearchURL(routeResult(nlu.IntentBusSearch, "تهران", "اصفهان", d))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// اصفهان must be "esfahan" for BUS (the spelling trap), and sort is required.
	want := "https://780.ir/tourism/bus/tehran-esfahan?departureDate=1405-05-21&sort=earliestTime"
	if got != want {
		t.Errorf("bus URL =\n  %s\nwant\n  %s", got, want)
	}
}

func TestBusIlam(t *testing.T) {
	// The bug that started all this: تهران→ایلام by bus must work.
	d := nlu.JalaliDate{Year: 1405, Month: 5, Day: 21}
	got, err := BuildBusSearchURL(routeResult(nlu.IntentBusSearch, "تهران", "ایلام", d))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://780.ir/tourism/bus/tehran-ilam?departureDate=1405-05-21&sort=earliestTime" {
		t.Errorf("bus to Ilam URL = %s", got)
	}
}

func TestTrainBusSpellingDiffers(t *testing.T) {
	// اهواز is ahvaz (train) vs ahwaz (bus) — different slug per service.
	if s, _ := LookupTrainCity("اهواز"); s != "ahvaz" {
		t.Errorf("train اهواز = %q, want ahvaz", s)
	}
	if s, _ := LookupBusCity("اهواز"); s != "ahwaz" {
		t.Errorf("bus اهواز = %q, want ahwaz", s)
	}
}

func TestTrainBusUnsupportedCity(t *testing.T) {
	d := nlu.JalaliDate{Year: 1405, Month: 5, Day: 20}
	// کیش is an island — no train/bus.
	if _, err := BuildTrainSearchURL(routeResult(nlu.IntentTrainSearch, "تهران", "کیش", d)); !errors.Is(err, ErrTrainCityUnsupported) {
		t.Errorf("train to Kish: want ErrTrainCityUnsupported, got %v", err)
	}
	if _, err := BuildBusSearchURL(routeResult(nlu.IntentBusSearch, "تهران", "کیش", d)); !errors.Is(err, ErrBusCityUnsupported) {
		t.Errorf("bus to Kish: want ErrBusCityUnsupported, got %v", err)
	}
	// اردبیل has bus but not train.
	if _, err := BuildTrainSearchURL(routeResult(nlu.IntentTrainSearch, "تهران", "اردبیل", d)); !errors.Is(err, ErrTrainCityUnsupported) {
		t.Errorf("train to Ardabil: want ErrTrainCityUnsupported, got %v", err)
	}
	if _, err := BuildBusSearchURL(routeResult(nlu.IntentBusSearch, "تهران", "اردبیل", d)); err != nil {
		t.Errorf("bus to Ardabil: unexpected error %v", err)
	}
}
