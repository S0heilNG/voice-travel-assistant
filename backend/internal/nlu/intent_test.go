package nlu

import "testing"

func TestDetectIntentPriority(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Intent
	}{
		// The core trap: "بلیط" (ticket) must not override an explicit mode.
		{"ticket+train -> train", "بلیط قطار تهران به مشهد فردا", IntentTrainSearch},
		{"ticket+bus -> bus", "بلیط اتوبوس تهران به اصفهان فردا", IntentBusSearch},
		{"bare train", "قطار تهران به مشهد فردا", IntentTrainSearch},
		{"bare bus", "اتوبوس تهران به اصفهان فردا", IntentBusSearch},
		{"train colloquial ترن (whole word)", "ترن تهران مشهد", IntentTrainSearch},
		// Prefix-path answers used during clarification.
		{"prefixed train nights-less answer", "قطار فردا", IntentTrainSearch},
		{"prefixed bus answer", "اتوبوس تهران", IntentBusSearch},
		// Regressions: flight and hotel unchanged.
		{"flight explicit", "بلیط هواپیما تهران به مشهد", IntentFlightSearch},
		{"flight via ticket+city", "بلیط تهران به مشهد فردا", IntentFlightSearch},
		{"flight city fallback", "می‌خوام برم کیش", IntentFlightSearch},
		{"hotel", "هتل مشهد", IntentHotelSearch},
		{"flight+hotel, flight first", "بلیط هواپیما و هتل تهران", IntentFlightSearch},
		{"flight+hotel, hotel first", "هتل و بلیط برای مشهد فردا", IntentHotelSearch},
		// Short-word safety: "اینترنت" contains "ترن", "اتوبوس" contains "بوس"
		// as substrings, but as whole-word keywords they must not misfire.
		{"internet is not train", "اینترنت خونه‌ام قطع شده", IntentUnknown},
		// Help / greeting — must be lowest priority.
		{"greeting -> help", "سلام حالت چطوره", IntentHelp},
		{"bare greeting -> help", "سلام", IntentHelp},
		{"capability question -> help", "چه کاری می‌تونی بکنی", IntentHelp},
		{"help keyword", "راهنما", IntentHelp},
		{"greeting + search stays flight", "سلام بلیط تهران به مشهد فردا", IntentFlightSearch},
		{"health is not a greeting", "سلامتی برات آرزو می‌کنم", IntentUnknown},
		{"help word loses to ticket+city", "کمکم کن بلیط تهران مشهد بخرم", IntentFlightSearch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectIntent(Normalize(tc.text)); got != tc.want {
				t.Errorf("detectIntent(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}
