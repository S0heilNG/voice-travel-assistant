package analytics

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestDisabledStoreIsSafeNoop(t *testing.T) {
	// Empty path -> disabled store. Every method must be a safe no-op.
	s, err := Open("")
	if err != nil {
		t.Fatalf("Open(\"\") returned error: %v", err)
	}
	if s.Enabled() {
		t.Error("empty-path store should be disabled")
	}
	// None of these may panic.
	s.LogParse(ParseLog{RawText: "x"})
	if s.LogEvent("sess", "search_clicked", "") {
		t.Error("disabled store should not accept events")
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close on disabled store: %v", err)
	}

	// A nil *Store must also be safe.
	var nilStore *Store
	nilStore.LogParse(ParseLog{})
	nilStore.LogEvent("s", "t", "")
	_ = nilStore.Close()
}

func TestLoggingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if !s.Enabled() {
		t.Fatal("store should be enabled")
	}

	s.LogParse(ParseLog{
		SessionID: "sess1", RawText: "بلیط تهران به مشهد", Intent: "flight_search",
		Origin: "تهران", Destination: "مشهد", Date: "1405-05-20",
		CitySupported: true, URLFromThisUtterance: true, Category: "",
	})
	s.LogParse(ParseLog{SessionID: "sess1", RawText: "تور کیش", Intent: "flight_search",
		Category: "unsupported_service", CategoryDetail: "tour"})

	if !s.LogEvent("sess1", "search_clicked", "") {
		t.Error("event should be accepted")
	}

	var parses, events int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM parses").Scan(&parses); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&events); err != nil {
		t.Fatal(err)
	}
	if parses != 2 || events != 1 {
		t.Errorf("counts = parses %d, events %d; want 2, 1", parses, events)
	}

	// Category column is queryable (the whole point of using SQLite).
	var raw string
	err = s.db.QueryRow(`SELECT raw_text FROM parses WHERE category='unsupported_service'`).Scan(&raw)
	if err != nil || raw != "تور کیش" {
		t.Errorf("category query got %q, err %v", raw, err)
	}
}

func TestTextTruncation(t *testing.T) {
	long := make([]byte, maxTextLen+50)
	for i := range long {
		long[i] = 'a'
	}
	if got := truncate(string(long)); len(got) != maxTextLen {
		t.Errorf("truncate len = %d, want %d", len(got), maxTextLen)
	}
}

func TestEventRateLimit(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	accepted := 0
	for i := 0; i < rateLimitPerWindow+20; i++ {
		if s.LogEvent("flooder", "click", "") {
			accepted++
		}
	}
	if accepted != rateLimitPerWindow {
		t.Errorf("accepted %d events, want cap of %d", accepted, rateLimitPerWindow)
	}
	// A different session is unaffected.
	if !s.LogEvent("other", "click", "") {
		t.Error("a different session should not be rate-limited")
	}
}

// A database created before the rename must keep working. Without the
// migration the INSERT would fail, and since LogParse swallows errors that
// would take analytics down silently rather than visibly.
func TestOpenMigratesLegacyColumnName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE parses (
		id INTEGER PRIMARY KEY AUTOINCREMENT, ts TEXT NOT NULL, session_id TEXT,
		raw_text TEXT, intent TEXT, origin TEXT, destination TEXT, date TEXT,
		nights INTEGER, missing TEXT, city_supported INTEGER,
		search_url_built INTEGER, category TEXT, category_detail TEXT)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on legacy db: %v", err)
	}
	defer s.Close()

	s.LogParse(ParseLog{SessionID: "sess1", RawText: "بلیط تهران به مشهد", URLFromThisUtterance: true})

	var got int
	if err := s.db.QueryRow(`SELECT url_from_this_utterance FROM parses`).Scan(&got); err != nil {
		t.Fatalf("read renamed column: %v", err)
	}
	if got != 1 {
		t.Errorf("url_from_this_utterance = %d, want 1", got)
	}
}
