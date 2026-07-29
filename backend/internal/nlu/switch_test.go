package nlu

import "testing"

// Mid-conversation, an utterance that names a service switches to it; one that
// doesn't keeps the running service. This is the layer the old keyword-prefix
// mechanism made impossible: prefixing "هتل" onto "قطار تهران به مشهد" put the
// hotel keyword first, and the earliest keyword wins, so the switch could
// never happen.
func TestServiceSwitchMidConversation(t *testing.T) {
	tests := []struct {
		name    string
		context Intent
		text    string
		want    Intent
	}{
		{"train named during a hotel flow", IntentHotelSearch, "قطار تهران به مشهد", IntentTrainSearch},
		{"hotel named during a flight flow", IntentFlightSearch, "هتل", IntentHotelSearch},
		{"tour named during a bus flow", IntentBusSearch, "تور کیش", IntentTourSearch},
		{"international named during a flight flow", IntentFlightSearch, "پرواز خارجی", IntentIntlFlightSearch},

		// Plain answers must NOT switch.
		{"bare date keeps train", IntentTrainSearch, "فردا", IntentTrainSearch},
		{"bare city keeps hotel", IntentHotelSearch, "مشهد", IntentHotelSearch},
		{"nights answer keeps hotel", IntentHotelSearch, "سه شب", IntentHotelSearch},

		// "بلیط" is generic and must never switch — it shows up in ordinary
		// replies and would derail them.
		{"ticket word keeps hotel", IntentHotelSearch, "بلیط فردا", IntentHotelSearch},
		{"ticket word keeps bus", IntentBusSearch, "بلیط تهران به مشهد", IntentBusSearch},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseWithContext(tc.text, tc.context).Intent; got != tc.want {
				t.Errorf("ParseWithContext(%q, ctx=%s) = %s, want %s", tc.text, tc.context, got, tc.want)
			}
		})
	}
}

// Without context, behaviour is exactly as before — a bare reply is not a
// search, and a full sentence still resolves on its own.
func TestParseWithoutContextUnchanged(t *testing.T) {
	if got := Parse("فردا").Intent; got == IntentTrainSearch {
		t.Errorf("bare date with no context = %s; context must not leak in", got)
	}
	if got := Parse("قطار تهران به قم فردا").Intent; got != IntentTrainSearch {
		t.Errorf("full train sentence = %s, want train_search", got)
	}
}

// A switch still extracts the new utterance's own entities.
func TestSwitchCarriesItsOwnEntities(t *testing.T) {
	got := ParseWithContext("قطار تهران به مشهد", IntentHotelSearch)
	if got.Intent != IntentTrainSearch {
		t.Fatalf("intent = %s, want train_search", got.Intent)
	}
	if got.Origin == nil || got.Origin.Name != "تهران" {
		t.Errorf("origin = %v, want تهران", got.Origin)
	}
	if got.Destination == nil || got.Destination.Name != "مشهد" {
		t.Errorf("destination = %v, want مشهد", got.Destination)
	}
}
