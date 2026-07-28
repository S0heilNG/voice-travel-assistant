package nlu

import "testing"

// These reuse fixedNow from nlu_test.go so every date in the package resolves
// against one deterministic clock.

func TestInternationalIntentFromCity(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantIntent Intent
		wantOrigin string
		wantDest   string
	}{
		{
			// The whole point: nothing says "خارجی", the destination decides.
			name: "foreign destination alone makes it international",
			text: "بلیط تهران به استانبول برای 15 مرداد", wantIntent: IntentIntlFlightSearch,
			wantOrigin: "IKA", wantDest: "IST",
		},
		{
			name: "foreign origin also makes it international",
			text: "پرواز از دبی به تهران فردا", wantIntent: IntentIntlFlightSearch,
			wantOrigin: "DXB", wantDest: "IKA",
		},
		{
			name: "non-Tehran Iranian origin keeps its own code",
			text: "پرواز مشهد به دبی فردا", wantIntent: IntentIntlFlightSearch,
			wantOrigin: "MHD", wantDest: "DXB",
		},
		{
			// Critical regression: an all-Iranian route stays domestic and
			// Tehran stays Mehrabad.
			name: "domestic route is untouched",
			text: "بلیط تهران به مشهد فردا", wantIntent: IntentFlightSearch,
			wantOrigin: "THR", wantDest: "MHD",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parse(tc.text, fixedNow)
			if got.Intent != tc.wantIntent {
				t.Fatalf("intent = %q, want %q", got.Intent, tc.wantIntent)
			}
			if got.Origin == nil || got.Origin.IATA != tc.wantOrigin {
				t.Errorf("origin = %+v, want IATA %q", got.Origin, tc.wantOrigin)
			}
			if got.Destination == nil || got.Destination.IATA != tc.wantDest {
				t.Errorf("destination = %+v, want IATA %q", got.Destination, tc.wantDest)
			}
		})
	}
}

func TestInternationalIntentFromKeyword(t *testing.T) {
	// An explicit keyword must work before any city is known, so the user is
	// asked for a route instead of being dropped into the domestic search.
	got := parse("پرواز خارجی می‌خوام", fixedNow)
	if got.Intent != IntentIntlFlightSearch {
		t.Fatalf("intent = %q, want %q", got.Intent, IntentIntlFlightSearch)
	}
	if len(got.Missing) == 0 {
		t.Error("expected missing fields for a bare international request")
	}
}

func TestRoundTrip(t *testing.T) {
	t.Run("date range yields a return date", func(t *testing.T) {
		got := parse("پرواز تهران به استانبول از 15 مرداد تا 22 مرداد", fixedNow)
		if got.Intent != IntentIntlFlightSearch {
			t.Fatalf("intent = %q", got.Intent)
		}
		if got.Date == nil || got.Date.String() != "1405-05-15" {
			t.Errorf("departure = %v, want 1405-05-15", got.Date)
		}
		if got.ReturnDate == nil || got.ReturnDate.String() != "1405-05-22" {
			t.Errorf("return = %v, want 1405-05-22", got.ReturnDate)
		}
	})

	t.Run("one-way when no return is mentioned", func(t *testing.T) {
		got := parse("بلیط تهران به استانبول برای 15 مرداد", fixedNow)
		if got.ReturnDate != nil {
			t.Errorf("return = %v, want nil (one-way)", got.ReturnDate)
		}
		for _, m := range got.Missing {
			if m == "returnDate" {
				t.Error("must not ask for a return date when none was requested")
			}
		}
	})

	t.Run("asks for a return date when round trip is requested without one", func(t *testing.T) {
		got := parse("بلیط رفت و برگشت تهران به استانبول برای 15 مرداد", fixedNow)
		found := false
		for _, m := range got.Missing {
			if m == "returnDate" {
				found = true
			}
		}
		if !found {
			t.Errorf("Missing = %v, want it to include returnDate", got.Missing)
		}
	})
}

// The vocabulary is 335 foreign names matched whole-word against free speech,
// so ordinary sentences must not become flight searches. These are the exact
// collisions that drove the exclusion list.
func TestAmbiguousWordsAreNotCities(t *testing.T) {
	tests := []string{
		"اسب رم کرد",
		"نیس بابا این چه حرفیه",
		"این کارو کن",
		"یه بار دیگه بگو",
		"وان حمام رو پر کن",
		"صدای بم داره",
		"پیج اینستاگرامش رو بده",
		"در امان باشی",
		"کبک تو باغه",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			got := parse(text, fixedNow)
			if got.Intent == IntentIntlFlightSearch {
				t.Errorf("parsed as international flight; origin=%v destination=%v", got.Origin, got.Destination)
			}
			if got.Destination != nil {
				t.Errorf("resolved a destination %q from an ordinary sentence", got.Destination.Name)
			}
		})
	}
}

// Multi-word and spelling variants must resolve to the same city.
func TestForeignAliases(t *testing.T) {
	tests := []struct{ text, wantIATA string }{
		{"اسلامبول", "IST"},
		{"استانبول", "IST"},
		{"کوالا لامپور", "KUL"},
		{"کوالالامپور", "KUL"},
		{"دهلی نو", "DEL"},
		{"تبلیسی", "TBS"},
		{"تفلیس", "TBS"},
		{"ابو ظبی", "AUH"},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			c := LookupCity(tc.text)
			if c == nil {
				t.Fatalf("LookupCity(%q) = nil", tc.text)
			}
			if c.IATA != tc.wantIATA {
				t.Errorf("IATA = %q, want %q", c.IATA, tc.wantIATA)
			}
			if !IsForeignCity(c) {
				t.Errorf("%q should be classified foreign", tc.text)
			}
		})
	}
}

// Iranian cities that also appear in 780's international map must stay
// domestic, or a Tehran→Mashhad search would flip to the international flow.
func TestIranianCitiesAreNotForeign(t *testing.T) {
	for _, name := range []string{"تهران", "مشهد", "شیراز", "کیش", "قشم", "رشت", "کرج", "یزد"} {
		c := LookupCity(name)
		if c == nil {
			t.Fatalf("LookupCity(%q) = nil", name)
		}
		if IsForeignCity(c) {
			t.Errorf("%q classified as foreign", name)
		}
	}
}
