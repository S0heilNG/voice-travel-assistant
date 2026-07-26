package nlu

import "testing"

// names extracts just the canonical city names a scan found, in order.
func foundCities(text string) []string {
	var out []string
	for _, m := range findCitiesInText(Normalize(text)) {
		out = append(out, m.city.Name)
	}
	return out
}

func TestFindCitiesWholeWord(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		// The classic overlap: کرمان is a substring of کرمانشاه but they are
		// different cities — whole-word matching must keep them apart.
		{"kermanshah is not kerman", "بلیط کرمانشاه", []string{"کرمانشاه"}},
		{"kerman alone", "بلیط کرمان", []string{"کرمان"}},
		// Multi-word alias, and its no-space spelling.
		{"bandar abbas two words", "از بندر عباس", []string{"بندرعباس"}},
		{"bandarabbas one word", "از بندرعباس", []string{"بندرعباس"}},
		// A real route with a newly added city (the reported bug).
		{"tehran to ilam", "اتوبوس تهران به ایلام", []string{"تهران", "ایلام"}},
		// Negatives: a city name embedded in a longer word must NOT match.
		{"karaji (boat) is not karaj", "یه کرجی تو دریا بود", nil},
		{"ahvazi (adjective) is not ahvaz", "من اهوازی هستم", nil},
		{"internet is not a city", "اینترنت خونه قطع شده", nil},
		{"no city", "هوا خیلی خوبه امروز", nil},
		// Deliberately-excluded common-word city names must NOT be recognized.
		{"bam (bass) excluded", "صدای بم دوست دارم", nil},
		{"khoy (temperament) excluded", "خوی خوبی داری", nil},
		{"siri (satiety) not a bare city", "آدم سیری ناپذیری هستم", nil},
		// Newly added cities across services.
		{"abadan airport city", "پرواز تهران به آبادان", []string{"تهران", "آبادان"}},
		{"qom bus/train-only city", "اتوبوس تهران به قم", []string{"تهران", "قم"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := foundCities(tc.text)
			if len(got) != len(tc.want) {
				t.Fatalf("foundCities(%q) = %v, want %v", tc.text, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("foundCities(%q) = %v, want %v", tc.text, got, tc.want)
					break
				}
			}
		})
	}
}

// The added cities must still parse into full route results.
func TestParseNewCityRoutes(t *testing.T) {
	got := parse("اتوبوس تهران به ایلام فردا", hotelNow)
	if got.Intent != IntentBusSearch {
		t.Errorf("intent = %q, want bus_search", got.Intent)
	}
	if got.Origin == nil || got.Origin.Name != "تهران" {
		t.Errorf("origin = %v, want تهران", got.Origin)
	}
	if got.Destination == nil || got.Destination.Name != "ایلام" {
		t.Errorf("destination = %v, want ایلام", got.Destination)
	}
	if len(got.Missing) != 0 {
		t.Errorf("missing = %v, want empty", got.Missing)
	}
}
