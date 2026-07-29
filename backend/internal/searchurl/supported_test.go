package searchurl

import (
	"strings"
	"testing"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"
)

func names(intents []nlu.Intent) string {
	parts := make([]string, len(intents))
	for i, in := range intents {
		parts[i] = string(in)
	}
	return strings.Join(parts, ",")
}

func TestSupportedServices(t *testing.T) {
	city := func(name string) nlu.City {
		if c := nlu.LookupCity(name); c != nil {
			return *c
		}
		t.Fatalf("unknown city %q", name)
		return nlu.City{}
	}

	tests := []struct {
		city string
		want string
	}{
		// An island: flyable, sleepable, tourable — but no rails or roads.
		{"کیش", "flight_search,hotel_search,tour_search"},
		// No airport, so only surface transport.
		{"قم", "train_search,bus_search"},
		// Big city, nearly everything.
		{"مشهد", "flight_search,train_search,bus_search,hotel_search,tour_search"},
	}
	for _, tc := range tests {
		if got := names(SupportedServices(city(tc.city))); got != tc.want {
			t.Errorf("SupportedServices(%s) = %s, want %s", tc.city, got, tc.want)
		}
	}

	// A foreign city is offered on the international service, never the
	// domestic one — no city is genuinely both.
	istanbul := city("استانبول")
	var sawDomestic, sawIntl bool
	for _, s := range SupportedServices(istanbul) {
		// Exact comparison, not substring: "flight_search" is a substring of
		// "international_flight_search".
		sawDomestic = sawDomestic || s == nlu.IntentFlightSearch
		sawIntl = sawIntl || s == nlu.IntentIntlFlightSearch
	}
	if sawDomestic || !sawIntl {
		t.Errorf("SupportedServices(استانبول) = %s, want international flight and not domestic",
			names(SupportedServices(istanbul)))
	}
}

func TestAlternativeServicesExcludesTheOneAsked(t *testing.T) {
	kish := *nlu.LookupCity("کیش")
	got := names(AlternativeServices(kish, nlu.IntentTrainSearch))
	if want := "flight_search,hotel_search,tour_search"; got != want {
		t.Errorf("alternatives for train to کیش = %s, want %s", got, want)
	}
	if strings.Contains(names(AlternativeServices(kish, nlu.IntentTourSearch)), "tour_search") {
		t.Error("alternatives still offered the service the user already asked for")
	}
}

// The early check must agree with what the URL builders actually do, or the
// user gets offered alternatives for a search that would have worked (or
// worse, no warning for one that wouldn't).
func TestCitySupportedForMatchesURLBuilders(t *testing.T) {
	date := &nlu.JalaliDate{Year: 1405, Month: 5, Day: 15}
	for _, name := range []string{"کیش", "قم", "مشهد", "تهران"} {
		c := nlu.LookupCity(name)
		if c == nil {
			t.Fatalf("unknown city %q", name)
		}
		for _, intent := range []nlu.Intent{
			nlu.IntentFlightSearch, nlu.IntentTrainSearch,
			nlu.IntentBusSearch, nlu.IntentHotelSearch, nlu.IntentTourSearch,
		} {
			// Same city at both ends isolates the single-city question.
			result := nlu.ParseResult{
				Intent: intent, Origin: c, Destination: c,
				Date: date, Nights: 2, Adults: 1,
			}
			var err error
			switch intent {
			case nlu.IntentFlightSearch:
				_, err = BuildFlightSearchURL(result)
			case nlu.IntentTrainSearch:
				_, err = BuildTrainSearchURL(result)
			case nlu.IntentBusSearch:
				_, err = BuildBusSearchURL(result)
			case nlu.IntentHotelSearch:
				_, err = BuildHotelSearchURL(result, 2)
			case nlu.IntentTourSearch:
				_, err = BuildTourSearchURL(result)
			}
			builderSaysOK := err == nil
			if got := CitySupportedFor(intent, *c); got != builderSaysOK {
				t.Errorf("%s/%s: CitySupportedFor=%v but builder err=%v", name, intent, got, err)
			}
		}
	}
}
