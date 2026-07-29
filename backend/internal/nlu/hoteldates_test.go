package nlu

import (
	"testing"
	"time"
)

// Fixed "now" for deterministic relative dates: 2026-07-01 is 1405-04-10
// (10 Tir 1405), a Wednesday.
var hotelNow = time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

func TestParseNights(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"digit", "3 شب", 3},
		{"digit no space", "2شب", 2},
		{"digit with colloquial he", "2 شبه", 2},
		{"persian digit (normalized)", Normalize("۵ شب"), 5},
		{"spelled se", "سه شب", 3},
		{"spelled yek", "یک شب", 1},
		{"spelled colloquial ye", "یه شب", 1},
		{"spelled dah", "ده شب", 10},
		{"spelled shesh", "شش شب", 6},
		{"in a sentence", Normalize("هتل مشهد ۴ شب می‌خوام"), 4},
		{"none", "هتل مشهد برای فردا", 0},
		{"amshab is not a nights count", "امشب می‌رسم", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseNights(tc.in); got != tc.want {
				t.Errorf("parseNights(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseHotelStay(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantCheck  *JalaliDate // nil = expect nil
		wantNights int
	}{
		{
			name:       "range with از/تا",
			in:         Normalize("هتل مشهد از ۲۰ مرداد تا ۲۵ مرداد"),
			wantCheck:  &JalaliDate{1405, 5, 20},
			wantNights: 5,
		},
		{
			name:       "range without از",
			in:         Normalize("هتل کیش ۲۰ مرداد تا ۲۳ مرداد"),
			wantCheck:  &JalaliDate{1405, 5, 20},
			wantNights: 3,
		},
		{
			name:       "range with الی",
			in:         Normalize("هتل شیراز از ۱ شهریور الی ۴ شهریور"),
			wantCheck:  &JalaliDate{1405, 6, 1},
			wantNights: 3,
		},
		{
			name:       "date plus nights count",
			in:         Normalize("هتل کیش ۳ شب از ۲۰ مرداد"),
			wantCheck:  &JalaliDate{1405, 5, 20},
			wantNights: 3,
		},
		{
			name:       "date only, no nights",
			in:         Normalize("هتل مشهد برای ۲۰ مرداد"),
			wantCheck:  &JalaliDate{1405, 5, 20},
			wantNights: 0,
		},
		{
			name:       "nights only, no date",
			in:         Normalize("هتل مشهد ۳ شب"),
			wantCheck:  nil,
			wantNights: 3,
		},
		{
			name:       "neither",
			in:         Normalize("هتل مشهد"),
			wantCheck:  nil,
			wantNights: 0,
		},
		{
			// 1405 is not a leap year, so Esfand has 29 days: 29 Esfand →
			// 1 Farvardin (next year) → 2 Farvardin = 2 nights. This exercises
			// the year-crossing bump (check-out resolves to 1406).
			name:       "range crossing the new year",
			in:         Normalize("هتل مشهد از ۲۹ اسفند تا ۲ فروردین"),
			wantCheck:  &JalaliDate{1405, 12, 29},
			wantNights: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCheck, gotNights := parseHotelStay(tc.in, hotelNow)
			if (gotCheck == nil) != (tc.wantCheck == nil) {
				t.Fatalf("checkIn nil mismatch: got %v, want %v", gotCheck, tc.wantCheck)
			}
			if gotCheck != nil && *gotCheck != *tc.wantCheck {
				t.Errorf("checkIn = %s, want %s", gotCheck, tc.wantCheck)
			}
			if gotNights != tc.wantNights {
				t.Errorf("nights = %d, want %d", gotNights, tc.wantNights)
			}
		})
	}
}

// End-to-end through parse(): hotel intent should surface Nights and the
// nights-missing state, and the prefix trick ("هتل ۳ شب") must still work.
func TestParseHotelNightsIntegration(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantNights  int
		wantMissing []string // subset check (must contain exactly these, order-insensitive)
	}{
		{"full via range", "هتل مشهد از ۲۰ مرداد تا ۲۵ مرداد", 5, nil},
		{"full via nights", "هتل مشهد ۳ شب برای ۲۰ مرداد", 3, nil},
		{"missing nights", "هتل مشهد برای ۲۰ مرداد", 0, []string{"nights"}},
		{"missing date and nights", "هتل مشهد", 0, []string{"date", "nights"}},
		// The clarification-answer prefix path: front-end sends "هتل" + answer.
		{"prefixed nights-only answer", "هتل ۳ شب", 3, []string{"destination", "date"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parse(tc.in, "", hotelNow)
			if got.Intent != IntentHotelSearch {
				t.Fatalf("intent = %q, want hotel_search", got.Intent)
			}
			if got.Nights != tc.wantNights {
				t.Errorf("Nights = %d, want %d", got.Nights, tc.wantNights)
			}
			for _, want := range tc.wantMissing {
				if !contains(got.Missing, want) {
					t.Errorf("Missing %v should contain %q", got.Missing, want)
				}
			}
			// "nights" must be reported missing iff Nights == 0.
			if (got.Nights == 0) != contains(got.Missing, "nights") {
				t.Errorf("nights-missing inconsistent: Nights=%d Missing=%v", got.Nights, got.Missing)
			}
		})
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
