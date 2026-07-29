package searchurl

// Tour destinations for 780.ir, from the platform's own zones endpoints
// (tour/v2/zones/DOMESTIC and .../INTERNATIONAL). The raw payloads are kept at
// docs/tour-zones-*.json: unlike every other vocabulary in this repo they are
// NOT re-derivable from anything public — the zones endpoints require
// authorization and no tour sitemap exists — so the source data is committed
// alongside the table it produced.
//
// ⚠️ Never invent a slug for a destination that isn't in this table. A wrong
// slug does not 404: 780's tour page hangs on an infinite loading spinner with
// no error at all, which is a worse dead end than the empty hotel page. An
// unknown destination must be reported as unsupported instead.
//
// The type is part of the data, not something to infer: it selects DOMESTIC vs
// INTERNATIONAL in the URL, and 780 groups a destination one way or the other
// itself (کربلا and نجف are INTERNATIONAL, as they should be).
//
// These lists are 780's *frequent* destinations, not an exhaustive catalogue —
// the endpoint field is literally "destinationFrequency". So coverage is 18
// destinations, and everything else falls through to the unsupported message.

// TourType is the value of the "type" query parameter.
type TourType string

const (
	TourDomestic      TourType = "DOMESTIC"
	TourInternational TourType = "INTERNATIONAL"
)

// TourDestination is one bookable tour destination.
type TourDestination struct {
	Slug string
	Type TourType
}

// tourDestinations is keyed by the canonical Persian name, matching the other
// per-service tables.
//
// Two entries from the source data are deliberately absent:
//   - "وان" (Van, Turkey) — also the everyday word for a bathtub and the
//     spoken form of "one". It is excluded from the international flight table
//     for exactly this reason; including it here would turn "وان حمام رو پر
//     کن" into a tour search.
//   - "پکن+شانگهای" — a combined two-city package whose name isn't a thing
//     anyone says. Mapping the spoken "پکن" to it would silently sell a
//     Beijing+Shanghai package to someone who asked about Beijing.
var tourDestinations = map[string]TourDestination{
	// DOMESTIC
	"مشهد":    {"mashhad", TourDomestic},
	"سرعین":   {"sareyn", TourDomestic},
	"ماسال":   {"masal", TourDomestic},
	"کیش":     {"kish", TourDomestic},
	"سوادکوه": {"savadkooh", TourDomestic},
	"شیرگاه":  {"shirgah", TourDomestic},
	"رشت":     {"rasht", TourDomestic},
	"رامسر":   {"ramsar", TourDomestic},
	"قزوین":   {"ghazvin", TourDomestic},
	"ارومیه":  {"orumie", TourDomestic},

	// INTERNATIONAL
	"استانبول": {"istanbul", TourInternational},
	"کربلا":    {"karbala", TourInternational},
	"آنتالیا":  {"antalya", TourInternational},
	"تفلیس":    {"tbilisi", TourInternational},
	"ایروان":   {"yerevan", TourInternational},
	"باتومی":   {"batumi", TourInternational},
	"نجف":      {"najaf", TourInternational},
	"دبی":      {"dubai", TourInternational},
}

// LookupTourDestination returns the tour destination for a canonical name.
func LookupTourDestination(name string) (TourDestination, bool) {
	d, ok := tourDestinations[name]
	return d, ok
}
