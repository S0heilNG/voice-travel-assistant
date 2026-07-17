package nlu

import (
	"reflect"
	"testing"
	"time"
)

// fixedNow is used everywhere a deterministic "now" is needed so date
// results are reproducible. 2026-07-17 is a Friday, which is 1405-04-26
// in the Jalali calendar.
var fixedNow = time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)

func jd(y, m, d int) *JalaliDate {
	return &JalaliDate{Year: y, Month: m, Day: d}
}

func city(name, iata string) *City {
	return &City{Name: name, IATA: iata}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantIntent  Intent
		wantOrigin  *City
		wantDest    *City
		wantDate    *JalaliDate
		wantMissing []string
	}{
		{
			name:        "flight with origin, destination and relative date",
			text:        "بلیط تهران به مشهد برای فردا",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 4, 27),
			wantMissing: nil,
		},
		{
			name:        "flight with از...به and پس‌فردا",
			text:        "پرواز از تهران به شیراز پس‌فردا",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("شیراز", "SYZ"),
			wantDate:    jd(1405, 4, 28),
			wantMissing: nil,
		},
		{
			name:        "no ticket keyword but known city — falls back to flight",
			text:        "می‌خوام برم کیش",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  nil,
			wantDest:    city("کیش", "KIH"),
			wantDate:    nil,
			wantMissing: []string{"origin", "date"},
		},
		{
			name:        "hotel with هفته بعد",
			text:        "هتل در مشهد برای هفته بعد",
			wantIntent:  IntentHotelSearch,
			wantOrigin:  nil,
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 5, 2),
			wantMissing: nil,
		},
		{
			name:        "flight, explicit date, no origin",
			text:        "بلیت هواپیما مشهد ۵ مرداد",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  nil,
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 5, 5),
			wantMissing: []string{"origin"},
		},
		{
			name:        "hotel via اتاق keyword, no date",
			text:        "یه اتاق تو اصفهان می‌خوام",
			wantIntent:  IntentHotelSearch,
			wantOrigin:  nil,
			wantDest:    city("اصفهان", "IFN"),
			wantDate:    nil,
			wantMissing: []string{"date"},
		},
		{
			name:        "unrelated sentence — unknown intent",
			text:        "هوا امروز چطوره",
			wantIntent:  IntentUnknown,
			wantOrigin:  nil,
			wantDest:    nil,
			wantDate:    nil,
			wantMissing: nil,
		},
		{
			name:        "Persian digits in explicit date",
			text:        "بلیط تهران به مشهد برای ۵ مرداد",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 5, 5),
			wantMissing: nil,
		},
		{
			name:        "Arabic yeh/kaf in the ticket keyword itself",
			text:        "بليط تهران به مشهد فردا", // بليط uses Arabic ي, not Persian ی
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 4, 27),
			wantMissing: nil,
		},
		{
			name:        "ZWNJ weekday name (سه‌شنبه) normalizes and resolves",
			text:        "بلیط تهران به تبریز سه‌شنبه",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("تبریز", "TBZ"),
			wantDate:    jd(1405, 4, 30),
			wantMissing: nil,
		},
		{
			name:        "solid-form پسفردا (no separator)",
			text:        "پرواز از تهران به قشم پسفردا",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("قشم", "GSM"),
			wantDate:    jd(1405, 4, 28),
			wantMissing: nil,
		},
		{
			name:        "multi-word city alias (بندر عباس)",
			text:        "بلیط از تهران به بندر عباس فردا",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("بندرعباس", "BND"),
			wantDate:    jd(1405, 4, 27),
			wantMissing: nil,
		},
		{
			name:        "امروز resolves to today",
			text:        "بلیط تهران به رشت امروز",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  city("تهران", "THR"),
			wantDest:    city("رشت", "RAS"),
			wantDate:    jd(1405, 4, 26),
			wantMissing: nil,
		},
		{
			name:        "hotel, single city, no در",
			text:        "هتل کرمان",
			wantIntent:  IntentHotelSearch,
			wantOrigin:  nil,
			wantDest:    city("کرمان", "KER"),
			wantDate:    nil,
			wantMissing: []string{"date"},
		},
		{
			name:        "both keywords present — flight keyword comes first",
			text:        "بلیط هواپیما و هتل تهران",
			wantIntent:  IntentFlightSearch,
			wantOrigin:  nil,
			wantDest:    city("تهران", "THR"),
			wantDate:    nil,
			wantMissing: []string{"origin", "date"},
		},
		{
			name:        "both keywords present — hotel keyword comes first",
			text:        "هتل و بلیط برای مشهد فردا",
			wantIntent:  IntentHotelSearch,
			wantOrigin:  nil,
			wantDest:    city("مشهد", "MHD"),
			wantDate:    jd(1405, 4, 27),
			wantMissing: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parse(tt.text, fixedNow)

			if got.Intent != tt.wantIntent {
				t.Errorf("Intent = %v, want %v", got.Intent, tt.wantIntent)
			}
			if !reflect.DeepEqual(got.Origin, tt.wantOrigin) {
				t.Errorf("Origin = %+v, want %+v", got.Origin, tt.wantOrigin)
			}
			if !reflect.DeepEqual(got.Destination, tt.wantDest) {
				t.Errorf("Destination = %+v, want %+v", got.Destination, tt.wantDest)
			}
			if !reflect.DeepEqual(got.Date, tt.wantDate) {
				t.Errorf("Date = %+v, want %+v", got.Date, tt.wantDate)
			}
			if !reflect.DeepEqual(got.Missing, tt.wantMissing) {
				t.Errorf("Missing = %v, want %v", got.Missing, tt.wantMissing)
			}
			if got.Adults != 1 {
				t.Errorf("Adults = %d, want 1", got.Adults)
			}
			if got.RawText != tt.text {
				t.Errorf("RawText = %q, want %q", got.RawText, tt.text)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name string
		text string
		want *JalaliDate
	}{
		{"today", "امروز می‌خوام برم", jd(1405, 4, 26)},
		{"tomorrow", "فردا می‌خوام برم", jd(1405, 4, 27)},
		{"day after tomorrow (zwnj form)", "پس‌فردا می‌خوام برم", jd(1405, 4, 28)},
		{"day after tomorrow (solid form)", "پسفردا می‌خوام برم", jd(1405, 4, 28)},
		{"next week", "هفته بعد می‌خوام برم", jd(1405, 5, 2)},
		{"next week (آینده)", "هفته آینده می‌خوام برم", jd(1405, 5, 2)},
		{"weekday شنبه", "شنبه می‌خوام برم", jd(1405, 4, 27)},
		{"weekday یکشنبه", "یکشنبه می‌خوام برم", jd(1405, 4, 28)},
		{"weekday دوشنبه", "دوشنبه می‌خوام برم", jd(1405, 4, 29)},
		{"weekday سه‌شنبه", "سه‌شنبه می‌خوام برم", jd(1405, 4, 30)},
		{"weekday چهارشنبه", "چهارشنبه می‌خوام برم", jd(1405, 4, 31)},
		{"weekday پنجشنبه", "پنجشنبه می‌خوام برم", jd(1405, 5, 1)},
		{"weekday جمعه — today is Friday, rolls to next week", "جمعه می‌خوام برم", jd(1405, 5, 2)},
		{"explicit date later this Jalali year", "۵ مرداد می‌خوام برم", jd(1405, 5, 5)},
		{"explicit date already passed this year rolls to next year", "۱۰ فروردین می‌خوام برم", jd(1406, 1, 10)},
		{"no date expression", "سلام حالت چطوره", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDate(Normalize(tt.text), fixedNow)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDate(%q) = %+v, want %+v", tt.text, got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"Arabic yeh/kaf converted", "بليط كيش", "بلیط کیش"},
		{"Persian digits converted", "۱۲۳", "123"},
		{"Arabic-Indic digits converted", "١٢٣", "123"},
		{"ZWNJ becomes space", "می‌خوام", "می خوام"},
		{"extra whitespace collapsed", "بلیط   تهران  به مشهد", "بلیط تهران به مشهد"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.text); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
