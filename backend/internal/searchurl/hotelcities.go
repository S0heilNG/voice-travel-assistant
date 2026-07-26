package searchurl

// 780.ir's hotel system identifies a city by a stable UUID, which is required
// on the search URL — without it the results page comes up empty. These UUIDs
// were captured from real searches on 780.ir and verified stable across
// repeated searches.
//
// This table deliberately lives here rather than in internal/nlu: a UUID is
// 780.ir integration data, not something about understanding Persian. Only
// these 8 cities have hotel coverage, so callers must handle the "known city,
// no hotel support" case.
type HotelCity struct {
	// NameEng is 780's English name, e.g. "Mashhad". It appears twice in the
	// URL: lowercased in the path, and as-is in the cityNameEng query param.
	NameEng string
	// ID is 780's hotel city UUID (the cityId param).
	ID string
}

// Keyed by the canonical Persian city name (nlu.City.Name) so it works for
// cities that have no airport/IATA. Every key must be a real nlu city name.
var hotelCities = map[string]HotelCity{
	"مشهد":  {NameEng: "Mashhad", ID: "ed48aafc-2d55-4a58-848c-4f0f7f0fd587"},
	"کیش":   {NameEng: "Kish", ID: "33efa9ce-ee1c-4e70-a612-abb2d5182c03"},
	"اصفهان": {NameEng: "Isfahan", ID: "84e89d24-f31f-40f2-aac2-bd03a9dc8c3c"},
	"شیراز":  {NameEng: "Shiraz", ID: "15c8c243-d23f-4a3d-8338-075fc5f7cfbb"},
	"تهران":  {NameEng: "Tehran", ID: "eb5f0fdb-c170-49a4-bab0-3e6ca0b12e03"},
	"تبریز":  {NameEng: "Tabriz", ID: "7d4d062d-a035-4278-b6f6-3e6b2c2ff925"},
	"یزد":    {NameEng: "Yazd", ID: "556b7d98-ca7f-44d8-9a1f-ed592d539338"},
	"قشم":    {NameEng: "Qeshm", ID: "299da048-9776-4ce9-b764-64a3de1497c9"},
}

// LookupHotelCity reports whether 780.ir supports hotel search for the given
// canonical Persian city name, and if so returns its hotel identity.
func LookupHotelCity(name string) (HotelCity, bool) {
	hc, ok := hotelCities[name]
	return hc, ok
}
