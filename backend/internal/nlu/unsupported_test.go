package nlu

import "testing"

func TestDetectUnsupportedService(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"ویلا شمال می‌خوام", "villa"},
		{"اقامتگاه بوم گردی", "villa"},
		// International flights and tours each became a supported service, so
		// neither may be reported as unmet demand any more.
		{"پرواز خارجی به استانبول", ""},
		{"بلیط بین المللی", ""},
		{"یه تور کیش می‌خوام", ""},
		{"تور استانبول", ""},
		// Negatives: supported requests must return "".
		{"بلیط تهران به مشهد فردا", ""},
		{"اتوبوس تهران به ایلام", ""},
		{"هتل مشهد", ""},
		{"قطار تهران به قم", ""},
		{"سلام", ""},
		// Whole-word safety: "تور" inside other words must not fire.
		{"دستور آشپزی می‌خوام", ""},
		{"موتور خراب شده", ""},
	}
	for _, tc := range tests {
		if got := DetectUnsupportedService(tc.text); got != tc.want {
			t.Errorf("DetectUnsupportedService(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

// Regression: the unsupported-service detector must not change how real
// requests are classified by detectIntent.
func TestUnsupportedDetectionDoesNotStealServices(t *testing.T) {
	tests := []struct {
		text string
		want Intent
	}{
		{"بلیط تهران به مشهد فردا", IntentFlightSearch},
		{"اتوبوس تهران به ایلام", IntentBusSearch},
		{"قطار تهران به قم", IntentTrainSearch},
		{"هتل مشهد", IntentHotelSearch},
		{"تور کیش", IntentTourSearch},
		{"سلام", IntentHelp},
	}
	for _, tc := range tests {
		if got := detectIntent(Normalize(tc.text)); got != tc.want {
			t.Errorf("detectIntent(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}
