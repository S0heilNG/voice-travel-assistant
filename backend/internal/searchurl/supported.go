package searchurl

import "github.com/soheilnegahi/voice-travel-assistant/backend/internal/nlu"

// SupportedServices reports every service 780.ir can search for a given city.
//
// This is the one place that answers "what CAN we do here?", and it is what
// turns a dead end into an offer: when the user asks for a train to کیش we can
// say what kیش *does* have instead of just refusing. Every per-service table is
// keyed by the same canonical Persian name, which is what makes the question
// answerable at all.
//
// Order is stable and roughly by usefulness, so the buttons the user sees don't
// jump around between requests.
func SupportedServices(city nlu.City) []nlu.Intent {
	var out []nlu.Intent

	// An Iranian city with a confirmed airport can be flown to domestically; a
	// foreign one is only reachable on the international service. Offering both
	// for the same city would be noise — no city is genuinely both.
	if nlu.IsForeignCity(&city) {
		out = append(out, nlu.IntentIntlFlightSearch)
	} else if city.IATA != "" {
		out = append(out, nlu.IntentFlightSearch)
	}

	if _, ok := LookupTrainCity(city.Name); ok {
		out = append(out, nlu.IntentTrainSearch)
	}
	if _, ok := LookupBusCity(city.Name); ok {
		out = append(out, nlu.IntentBusSearch)
	}
	if _, ok := LookupHotelCity(city.Name); ok {
		out = append(out, nlu.IntentHotelSearch)
	}
	if _, ok := LookupTourDestination(city.Name); ok {
		out = append(out, nlu.IntentTourSearch)
	}
	return out
}

// CitySupportedFor reports whether one city works for one service. It is the
// same question BuildXSearchURL answers, but askable before the request is
// complete — which matters: "قطار کیش" should say Kish has no train right
// away, not after making the user answer "from where?" first and only then
// dead-ending them.
func CitySupportedFor(intent nlu.Intent, city nlu.City) bool {
	switch intent {
	case nlu.IntentFlightSearch, nlu.IntentIntlFlightSearch:
		return city.IATA != ""
	case nlu.IntentTrainSearch:
		_, ok := LookupTrainCity(city.Name)
		return ok
	case nlu.IntentBusSearch:
		_, ok := LookupBusCity(city.Name)
		return ok
	case nlu.IntentHotelSearch:
		_, ok := LookupHotelCity(city.Name)
		return ok
	case nlu.IntentTourSearch:
		_, ok := LookupTourDestination(city.Name)
		return ok
	}
	return true
}

// AlternativeServices is SupportedServices minus the one the user already
// asked for — i.e. exactly what to offer them instead. Returns nil when there
// is nothing else, so the caller can fall back to a generic message rather than
// rendering an empty button row.
func AlternativeServices(city nlu.City, asked nlu.Intent) []nlu.Intent {
	var out []nlu.Intent
	for _, s := range SupportedServices(city) {
		if s != asked {
			out = append(out, s)
		}
	}
	return out
}
