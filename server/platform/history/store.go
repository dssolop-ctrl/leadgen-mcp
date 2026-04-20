// Package history persists a centralized change log for ad-cabinet operations.
//
// Two layers live in the same SQLite DB:
//
//   - change_events: append-only per-mutation records (who/when/what/before/after).
//   - daily_summaries: one row per (agency_account, city_login, YYYY-MM-DD), UPDATEd
//     throughout the day. New day → new row; yesterday is never overwritten.
//
// The tree key (agency_account → city_login → campaign_id → event) is implicit in
// the columns; queries filter by any combination of those fields.
package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite handle for history data.
//
// Persistence uses journal_mode=delete + synchronous=full — same tradeoff as
// the filters store: one extra fsync per commit, but no WAL lost-on-SIGKILL
// risk on Docker bind-mounts.
type Store struct {
	db *sql.DB
}

// ChangeEvent is a single audited mutation recorded by the agent.
type ChangeEvent struct {
	ID             int64  `json:"id,omitempty"`
	Timestamp      string `json:"timestamp"`        // RFC3339, UTC
	AgencyAccount  string `json:"agency_account"`   // e.g. "etagi click"
	CityLogin      string `json:"city_login"`       // Direct client-login
	City           string `json:"city,omitempty"`   // human slug: omsk, spb
	CampaignID     string `json:"campaign_id,omitempty"`
	CampaignName   string `json:"campaign_name,omitempty"`
	EntityType     string `json:"entity_type"` // campaign|adgroup|ad|keyword|negative|bid|strategy|budget|target
	EntityID       string `json:"entity_id,omitempty"`
	ActionType     string `json:"action_type"` // create|update|pause|resume|archive|delete|moderate
	ToolName       string `json:"tool_name,omitempty"`
	BeforeValue    string `json:"before_value,omitempty"` // JSON snippet or short text
	AfterValue     string `json:"after_value,omitempty"`
	Reason         string `json:"reason,omitempty"`
	OperatorNote   string `json:"operator_note,omitempty"`
	CorrelationKey string `json:"correlation_key,omitempty"` // ties events from one task together
}

// DailySummary is one aggregated note per city per day.
type DailySummary struct {
	ID            int64  `json:"id,omitempty"`
	Date          string `json:"date"` // YYYY-MM-DD (UTC day, see daySlug)
	AgencyAccount string `json:"agency_account"`
	CityLogin     string `json:"city_login"`
	City          string `json:"city,omitempty"`
	Summary       string `json:"summary"`
	OperatorName  string `json:"operator_name,omitempty"`
	EventCount    int    `json:"event_count"`
	UpdatedAt     string `json:"updated_at"` // RFC3339
}

// Open opens or creates the history DB at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(delete)&_pragma=synchronous(full)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying DB handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS change_events (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp        TEXT NOT NULL,
			agency_account   TEXT NOT NULL DEFAULT '',
			city_login       TEXT NOT NULL DEFAULT '',
			city             TEXT NOT NULL DEFAULT '',
			campaign_id      TEXT NOT NULL DEFAULT '',
			campaign_name    TEXT NOT NULL DEFAULT '',
			entity_type      TEXT NOT NULL,
			entity_id        TEXT NOT NULL DEFAULT '',
			action_type      TEXT NOT NULL,
			tool_name        TEXT NOT NULL DEFAULT '',
			before_value     TEXT NOT NULL DEFAULT '',
			after_value      TEXT NOT NULL DEFAULT '',
			reason           TEXT NOT NULL DEFAULT '',
			operator_note    TEXT NOT NULL DEFAULT '',
			correlation_key  TEXT NOT NULL DEFAULT ''
		);

		CREATE INDEX IF NOT EXISTS idx_ce_city_login ON change_events(city_login);
		CREATE INDEX IF NOT EXISTS idx_ce_campaign   ON change_events(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_ce_ts         ON change_events(timestamp);
		CREATE INDEX IF NOT EXISTS idx_ce_corr       ON change_events(correlation_key);

		CREATE TABLE IF NOT EXISTS daily_summaries (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			date            TEXT NOT NULL,
			agency_account  TEXT NOT NULL DEFAULT '',
			city_login      TEXT NOT NULL DEFAULT '',
			city            TEXT NOT NULL DEFAULT '',
			summary         TEXT NOT NULL DEFAULT '',
			operator_name   TEXT NOT NULL DEFAULT '',
			event_count     INTEGER NOT NULL DEFAULT 0,
			updated_at      TEXT NOT NULL,
			UNIQUE(date, city_login)
		);

		CREATE INDEX IF NOT EXISTS idx_ds_city       ON daily_summaries(city_login);
		CREATE INDEX IF NOT EXISTS idx_ds_date       ON daily_summaries(date);
	`)
	return err
}

// --- change_events ---

// LogEvent appends a change event. Timestamp is auto-set to UTC now if empty.
func (s *Store) LogEvent(e ChangeEvent) (int64, error) {
	if strings.TrimSpace(e.EntityType) == "" || strings.TrimSpace(e.ActionType) == "" {
		return 0, fmt.Errorf("entity_type and action_type are required")
	}
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := s.db.Exec(`
		INSERT INTO change_events
		(timestamp, agency_account, city_login, city, campaign_id, campaign_name,
		 entity_type, entity_id, action_type, tool_name, before_value, after_value,
		 reason, operator_note, correlation_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp, e.AgencyAccount, e.CityLogin, e.City, e.CampaignID, e.CampaignName,
		e.EntityType, e.EntityID, e.ActionType, e.ToolName, e.BeforeValue, e.AfterValue,
		e.Reason, e.OperatorNote, e.CorrelationKey)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// QueryEvents filters change events by any combination of fields.
// Empty filter fields are ignored. dateFrom/dateTo accept YYYY-MM-DD or RFC3339.
func (s *Store) QueryEvents(campaignID, cityLogin, agency, dateFrom, dateTo, correlationKey string, limit int) ([]ChangeEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var (
		where []string
		args  []any
	)
	if campaignID != "" {
		where = append(where, "campaign_id = ?")
		args = append(args, campaignID)
	}
	if cityLogin != "" {
		where = append(where, "city_login = ?")
		args = append(args, cityLogin)
	}
	if agency != "" {
		where = append(where, "agency_account = ?")
		args = append(args, agency)
	}
	if dateFrom != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, normalizeDate(dateFrom, false))
	}
	if dateTo != "" {
		where = append(where, "timestamp <= ?")
		args = append(args, normalizeDate(dateTo, true))
	}
	if correlationKey != "" {
		where = append(where, "correlation_key = ?")
		args = append(args, correlationKey)
	}

	q := `SELECT id, timestamp, agency_account, city_login, city, campaign_id, campaign_name,
		entity_type, entity_id, action_type, tool_name, before_value, after_value,
		reason, operator_note, correlation_key
		FROM change_events`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChangeEvent
	for rows.Next() {
		var e ChangeEvent
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.AgencyAccount, &e.CityLogin, &e.City,
			&e.CampaignID, &e.CampaignName, &e.EntityType, &e.EntityID, &e.ActionType,
			&e.ToolName, &e.BeforeValue, &e.AfterValue, &e.Reason, &e.OperatorNote,
			&e.CorrelationKey); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// --- daily_summaries ---

// UpsertDailySummary creates or updates today's summary for the given city/login pair.
// `date` is YYYY-MM-DD; if empty, uses UTC today. UPDATE on conflict keeps the row for
// that day accumulating; a new day creates a new row without touching yesterday's.
// If `appendText` is true, newSummary is appended to the existing summary (with newline).
// Otherwise the existing summary is replaced.
func (s *Store) UpsertDailySummary(d DailySummary, appendText bool) (DailySummary, error) {
	if d.Date == "" {
		d.Date = time.Now().UTC().Format("2006-01-02")
	}
	if strings.TrimSpace(d.CityLogin) == "" {
		return DailySummary{}, fmt.Errorf("city_login is required")
	}
	d.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Read existing (if any) to support append + count preservation.
	var existing DailySummary
	row := s.db.QueryRow(`SELECT id, summary, event_count FROM daily_summaries WHERE date = ? AND city_login = ?`,
		d.Date, d.CityLogin)
	_ = row.Scan(&existing.ID, &existing.Summary, &existing.EventCount)

	newSummary := d.Summary
	if appendText && existing.Summary != "" && newSummary != "" {
		newSummary = existing.Summary + "\n" + newSummary
	} else if appendText && newSummary == "" {
		newSummary = existing.Summary
	}

	if existing.ID > 0 {
		_, err := s.db.Exec(`
			UPDATE daily_summaries
			SET summary = ?, operator_name = COALESCE(NULLIF(?, ''), operator_name),
			    event_count = event_count + ?, updated_at = ?, agency_account = ?, city = ?
			WHERE id = ?`,
			newSummary, d.OperatorName, d.EventCount, d.UpdatedAt, d.AgencyAccount, d.City, existing.ID)
		if err != nil {
			return DailySummary{}, err
		}
		d.ID = existing.ID
		d.Summary = newSummary
		d.EventCount = existing.EventCount + d.EventCount
		return d, nil
	}

	res, err := s.db.Exec(`
		INSERT INTO daily_summaries
		(date, agency_account, city_login, city, summary, operator_name, event_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.Date, d.AgencyAccount, d.CityLogin, d.City, newSummary, d.OperatorName, d.EventCount, d.UpdatedAt)
	if err != nil {
		return DailySummary{}, err
	}
	d.ID, _ = res.LastInsertId()
	d.Summary = newSummary
	return d, nil
}

// GetDailySummaries returns summaries in descending date order, filtered by city and window.
// Limit defaults to 30 rows.
func (s *Store) GetDailySummaries(cityLogin, dateFrom, dateTo string, limit int) ([]DailySummary, error) {
	if limit <= 0 || limit > 365 {
		limit = 30
	}
	var (
		where []string
		args  []any
	)
	if cityLogin != "" {
		where = append(where, "city_login = ?")
		args = append(args, cityLogin)
	}
	if dateFrom != "" {
		where = append(where, "date >= ?")
		args = append(args, normalizeDate(dateFrom, false)[:10])
	}
	if dateTo != "" {
		where = append(where, "date <= ?")
		args = append(args, normalizeDate(dateTo, false)[:10])
	}
	q := `SELECT id, date, agency_account, city_login, city, summary, operator_name, event_count, updated_at
		FROM daily_summaries`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY date DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailySummary
	for rows.Next() {
		var d DailySummary
		if err := rows.Scan(&d.ID, &d.Date, &d.AgencyAccount, &d.CityLogin, &d.City, &d.Summary,
			&d.OperatorName, &d.EventCount, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Stats returns quick counts for health checks.
func (s *Store) Stats() (map[string]int, error) {
	out := map[string]int{}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM change_events`).Scan(&n); err != nil {
		return nil, err
	}
	out["events_total"] = n
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM daily_summaries`).Scan(&n); err != nil {
		return nil, err
	}
	out["daily_summaries_total"] = n
	return out, nil
}

// normalizeDate accepts YYYY-MM-DD or RFC3339 and returns an RFC3339 string.
// When endOfDay is true and input is date-only, returns end-of-day (23:59:59Z).
func normalizeDate(s string, endOfDay bool) string {
	s = strings.TrimSpace(s)
	if len(s) == 10 {
		if endOfDay {
			return s + "T23:59:59Z"
		}
		return s + "T00:00:00Z"
	}
	return s
}
