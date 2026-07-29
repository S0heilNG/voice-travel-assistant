package nlu

import "testing"

func TestTourIntentAndDestination(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantDest string // "" means none resolved
		wantMiss []string
	}{
		{"domestic destination", "تور کیش", "کیش", nil},
		{"international destination", "تور استانبول", "استانبول", nil},
		{"tour-only destination", "تور سرعین می‌خوام", "سرعین", nil},
		{"multi-word alias", "تور جزیره کیش", "کیش", nil},
		{"no destination yet", "تور می‌خوام", "", []string{"destination"}},
		// A known city with no tour still resolves, so the URL builder can say
		// "no tour there" instead of the assistant re-asking a question the
		// user already answered.
		{"known city without a tour", "تور تهران", "تهران", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.text)
			if got.Intent != IntentTourSearch {
				t.Fatalf("intent = %q, want %q", got.Intent, IntentTourSearch)
			}
			gotDest := ""
			if got.Destination != nil {
				gotDest = got.Destination.Name
			}
			if gotDest != tc.wantDest {
				t.Errorf("destination = %q, want %q", gotDest, tc.wantDest)
			}
			if len(got.Missing) != len(tc.wantMiss) {
				t.Fatalf("missing = %v, want %v", got.Missing, tc.wantMiss)
			}
			for i := range tc.wantMiss {
				if got.Missing[i] != tc.wantMiss[i] {
					t.Errorf("missing = %v, want %v", got.Missing, tc.wantMiss)
				}
			}
		})
	}
}

// Tours need neither an origin nor a date, so neither may ever be reported as
// missing — that is what makes "تور کیش" a complete request in one word.
func TestTourNeverAsksForOriginOrDate(t *testing.T) {
	for _, text := range []string{"تور کیش", "تور می‌خوام", "تور استانبول"} {
		for _, field := range Parse(text).Missing {
			if field == "origin" || field == "date" || field == "nights" {
				t.Errorf("Parse(%q) reported missing %q; tours have no such slot", text, field)
			}
		}
	}
}

// "تور" is a whole word here, so it must not fire inside longer words. The
// non-travel senses of the standalone word ("تور والیبال") are a documented,
// accepted risk and are asserted separately below.
func TestTourWordBoundaries(t *testing.T) {
	for _, text := range []string{
		"دستور آشپزی می‌خوام",
		"موتور خراب شده",
		"کنسرتور رو بیار",
	} {
		if got := Parse(text).Intent; got == IntentTourSearch {
			t.Errorf("Parse(%q) = tour_search; \"تور\" matched inside another word", text)
		}
	}
}

// Documents the accepted false-positive: the standalone word "تور" has
// non-travel senses. They do parse as tour searches, but with no bookable
// destination the assistant only asks which destination — it never invents one.
func TestTourNonTravelSensesAreBounded(t *testing.T) {
	for _, text := range []string{"تور والیبال خریدم", "تور عروس قشنگه", "تور ماهیگیری"} {
		got := Parse(text)
		if got.Intent != IntentTourSearch {
			continue // even better, but not what we assert
		}
		if got.Destination != nil {
			t.Errorf("Parse(%q) invented destination %q", text, got.Destination.Name)
		}
		if len(got.Missing) == 0 {
			t.Errorf("Parse(%q) looked complete; it must ask for a destination", text)
		}
	}
}

// Every tour destination the NLU can resolve must exist in the URL table, or a
// user could be told a tour exists and then get no link.
func TestTourVocabularyNamesAreCanonical(t *testing.T) {
	for _, entry := range tourDestEntries {
		for _, alias := range entry.aliases {
			got := extractTourDestination(Normalize("تور " + alias))
			if got == nil {
				t.Errorf("alias %q of %q did not resolve", alias, entry.city.Name)
				continue
			}
			if got.Name != entry.city.Name {
				t.Errorf("alias %q resolved to %q, want %q", alias, got.Name, entry.city.Name)
			}
		}
	}
}
