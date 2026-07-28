// Package analytics records anonymous interaction data during the limited-user
// test so we can learn what users actually want — especially the raw text of
// requests the assistant couldn't serve, grouped into categories.
//
// Design rules that must hold:
//   - Logging never breaks a user request. Every write swallows its error (logs
//     it server-side only). A nil *Store, or one that failed to open, is a
//     safe no-op — the app works fully without a database.
//   - No personal data. The only identifier stored is a random session id the
//     frontend generates; raw text is truncated.
//
// The database file lives OUTSIDE the release directory (in the deploy's
// shared/ dir, next to .env), so deploy.sh's release rotation never discards
// it. The path comes from the VTA_ANALYTICS_DB env var; when empty, logging is
// disabled entirely.
package analytics

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo)
)

// maxTextLen caps stored raw text so a hostile client can't bloat the DB.
const maxTextLen = 500

// Store persists parse logs and funnel events. All methods are safe to call on
// a nil receiver or a Store whose db failed to open.
type Store struct {
	db *sql.DB

	// Simple per-session rate limiter for the events endpoint.
	mu      sync.Mutex
	windows map[string]*rateWindow
}

type rateWindow struct {
	count int
	start time.Time
}

const (
	rateLimitPerWindow = 120
	rateWindow_        = time.Minute
)

// Open opens (and migrates) the SQLite database at path. When path is empty it
// returns a disabled no-op Store and nil error — logging is simply off. A real
// open error is returned so the caller can log it, but the caller should still
// run with the (disabled) Store rather than crash.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		log.Printf("analytics: no VTA_ANALYTICS_DB set — interaction logging disabled")
		return &Store{}, nil
	}

	// _busy_timeout so concurrent writes wait rather than error; WAL for better
	// read/write concurrency during analysis via a separate sqlite3 session.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return &Store{}, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return &Store{}, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return &Store{}, err
	}
	log.Printf("analytics: logging to %s", path)
	return &Store{db: db, windows: map[string]*rateWindow{}}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS parses (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ts              TEXT NOT NULL,
    session_id      TEXT,
    raw_text        TEXT,
    intent          TEXT,
    origin          TEXT,
    destination     TEXT,
    date            TEXT,
    nights          INTEGER,
    missing         TEXT,
    city_supported  INTEGER,
    url_from_this_utterance INTEGER,
    category        TEXT,
    category_detail TEXT
);
CREATE INDEX IF NOT EXISTS idx_parses_category ON parses(category);
CREATE INDEX IF NOT EXISTS idx_parses_session ON parses(session_id);

CREATE TABLE IF NOT EXISTS events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    ts          TEXT NOT NULL,
    session_id  TEXT,
    type        TEXT NOT NULL,
    detail      TEXT
);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
`)
	if err != nil {
		return err
	}
	return renameLegacySearchURLColumn(db)
}

// renameLegacySearchURLColumn brings pre-existing databases up to the current
// column name. CREATE TABLE IF NOT EXISTS leaves an older table untouched, so
// without this an existing file would keep the old column and every INSERT
// would fail — and because LogParse swallows its errors, analytics would go
// silently dead rather than loudly broken.
//
// The old name, "search_url_built", read as "the user completed a search". It
// never meant that: the backend parses each utterance independently, so in any
// multi-turn conversation the final utterance carries only the last slot and
// the column is 0 even when the user did go on to search. See docs/analytics.md.
func renameLegacySearchURLColumn(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('parses')`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var hasLegacy, hasCurrent bool
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		switch name {
		case "search_url_built":
			hasLegacy = true
		case "url_from_this_utterance":
			hasCurrent = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasLegacy || hasCurrent {
		return nil
	}
	_, err = db.Exec(`ALTER TABLE parses RENAME COLUMN search_url_built TO url_from_this_utterance`)
	return err
}

// Close closes the database if it's open.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Enabled reports whether logging is actually happening.
func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func truncate(s string) string {
	if len(s) > maxTextLen {
		return s[:maxTextLen]
	}
	return s
}

// ParseLog is one recorded /api/parse call.
type ParseLog struct {
	SessionID     string
	RawText       string
	Intent        string
	Origin        string
	Destination   string
	Date          string
	Nights        int
	Missing       string
	CitySupported bool
	// URLFromThisUtterance records whether *this single utterance* alone
	// produced a search URL. It is not a success metric: slots accumulate in
	// the frontend, so a multi-turn conversation logs 0 here even when the
	// user searched. The success signal is the search_clicked event.
	URLFromThisUtterance bool
	Category             string
	CategoryDetail       string
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// LogParse records a parse. Errors are swallowed (logged only) so a logging
// failure can never turn into a failed user request.
func (s *Store) LogParse(p ParseLog) {
	if !s.Enabled() {
		return
	}
	_, err := s.db.Exec(
		`INSERT INTO parses
		 (ts, session_id, raw_text, intent, origin, destination, date, nights,
		  missing, city_supported, url_from_this_utterance, category, category_detail)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		time.Now().UTC().Format(time.RFC3339), p.SessionID, truncate(p.RawText),
		p.Intent, p.Origin, p.Destination, p.Date, p.Nights, p.Missing,
		b2i(p.CitySupported), b2i(p.URLFromThisUtterance), p.Category, p.CategoryDetail,
	)
	if err != nil {
		log.Printf("analytics: LogParse failed: %v", err)
	}
}

// LogEvent records a funnel event. Returns whether it was accepted (false when
// rate-limited or disabled) so the handler can respond, but a false return is
// never an error the client needs to act on.
func (s *Store) LogEvent(sessionID, eventType, detail string) bool {
	if !s.Enabled() {
		return false
	}
	if !s.allow(sessionID) {
		return false
	}
	_, err := s.db.Exec(
		`INSERT INTO events (ts, session_id, type, detail) VALUES (?,?,?,?)`,
		time.Now().UTC().Format(time.RFC3339), sessionID, eventType, truncate(detail),
	)
	if err != nil {
		log.Printf("analytics: LogEvent failed: %v", err)
		return false
	}
	return true
}

// allow is a small fixed-window rate limiter, per session, for the public
// events endpoint.
func (s *Store) allow(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	w := s.windows[sessionID]
	if w == nil || now.Sub(w.start) > rateWindow_ {
		s.windows[sessionID] = &rateWindow{count: 1, start: now}
		return true
	}
	if w.count >= rateLimitPerWindow {
		return false
	}
	w.count++
	return true
}
