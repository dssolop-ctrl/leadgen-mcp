package filters

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Store manages site filter data in SQLite.
//
// Persistence design:
//   - SQLite journal_mode=delete + synchronous=full — each commit hits disk, no WAL
//     lost-on-SIGKILL risk. This trades a bit of write throughput for robustness on
//     Docker bind-mounts (Windows/macOS WSL2), which is the target deployment.
//   - If exportPath is set, Open seeds an empty DB from that JSON file on first start
//     (git-clone recovery path) and re-writes the file after every user write so the
//     repo-tracked seed stays in sync with live data.
type Store struct {
	db         *sql.DB
	exportPath string     // path to JSON seed / auto-export file; "" disables export.
	exportMu   sync.Mutex // serializes atomic JSON writes.
}

// Open opens or creates the SQLite database at the given path.
// If exportPath is non-empty:
//   - on empty DB (filter_values count == 0), server bulk-loads filter_values from it;
//   - on every user write, server dumps current filter_values back to this path.
// Seeding failures are logged but do not fail startup.
func Open(path, exportPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	// journal_mode=delete avoids WAL/-shm files, which behave poorly over Docker
	// bind mounts on Windows/macOS. synchronous=full fsyncs each commit.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(delete)&_pragma=synchronous(full)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db, exportPath: exportPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := s.loadSeedIfEmpty(); err != nil {
		// Non-fatal — log and continue with an empty DB.
		log.Printf("filters: seed load warning: %v", err)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_types (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL UNIQUE,     -- "rooms", "district", "price_from"
			url_param  TEXT NOT NULL,             -- URL query parameter name
			value_type TEXT NOT NULL DEFAULT 'string' -- "int", "string", "bool"
		);

		CREATE TABLE IF NOT EXISTS filter_values (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			city_slug      TEXT NOT NULL,            -- "omsk", "spb", "krasnodar"
			filter_type_id INTEGER NOT NULL REFERENCES filter_types(id),
			name           TEXT NOT NULL,            -- human name: "Нефтяники", "1-комнатная"
			url_value      TEXT NOT NULL,            -- value for URL param: "42", "1"
			aliases        TEXT NOT NULL DEFAULT '',  -- comma-separated aliases for fuzzy match
			UNIQUE(city_slug, filter_type_id, url_value)
		);

		CREATE TABLE IF NOT EXISTS url_templates (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			theme     TEXT NOT NULL UNIQUE,  -- "вторичка", "загородка", "новостройки"
			base_path TEXT NOT NULL          -- "/realty/", "/realty_out/", "/zastr/"
		);

		CREATE INDEX IF NOT EXISTS idx_fv_city ON filter_values(city_slug);
		CREATE INDEX IF NOT EXISTS idx_fv_type ON filter_values(filter_type_id);
	`)
	if err != nil {
		return err
	}
	// Seed or update default filter types.
	for _, ft := range defaultFilterTypes {
		s.db.Exec(`INSERT INTO filter_types (name, url_param, value_type) VALUES (?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET url_param=excluded.url_param, value_type=excluded.value_type`,
			ft.Name, ft.URLParam, ft.ValueType)
	}
	// Seed or update default URL templates.
	for _, t := range defaultTemplates {
		s.db.Exec(`INSERT INTO url_templates (theme, base_path) VALUES (?, ?)
			ON CONFLICT(theme) DO UPDATE SET base_path=excluded.base_path`,
			t.Theme, t.BasePath)
	}
	return nil
}

// --- Data types ---

// FilterType describes a type of URL filter.
type FilterType struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	URLParam  string `json:"url_param"`
	ValueType string `json:"value_type"`
}

// FilterValue is a specific filter value for a city.
type FilterValue struct {
	ID           int    `json:"id,omitempty"`
	CitySlug     string `json:"city_slug"`
	FilterType   string `json:"filter_type"`
	Name         string `json:"name"`
	URLValue     string `json:"url_value"`
	Aliases      string `json:"aliases,omitempty"`
}

// URLTemplate maps a theme to a base URL path.
type URLTemplate struct {
	Theme    string `json:"theme"`
	BasePath string `json:"base_path"`
}

// --- Default seeds ---

var defaultFilterTypes = []FilterType{
	// Rooms: rooms[]=1, rooms[]=2, rooms[]=3, rooms[]=>4; studio: studio[]=true
	{Name: "rooms", URLParam: "rooms[]", ValueType: "int"},
	{Name: "studio", URLParam: "studio[]", ValueType: "string"},
	// Geo: district_id[]=82206
	{Name: "district", URLParam: "district_id[]", ValueType: "id"},
	// Price: price_min=15, price_max=55 (in some unit, probably thousands)
	{Name: "price_min", URLParam: "price_min", ValueType: "int"},
	{Name: "price_max", URLParam: "price_max", ValueType: "int"},
	// Area: square_min=15, square_max=25 (sq.m.)
	{Name: "square_min", URLParam: "square_min", ValueType: "int"},
	{Name: "square_max", URLParam: "square_max", ValueType: "int"},
	// Property type (вторичка): type[]=1 (квартира), 2 (пансионат), 3 (малосемейка), 4 (общежитие), 6 (апартаменты), 8 (гостинка)
	{Name: "property_type", URLParam: "type[]", ValueType: "id"},
	// Property type (загородка): m_obj_type[]=12 (участок), 13 (дом), 14 (дача), 15 (таунхаус), 16 (половина дома)
	{Name: "property_type_out", URLParam: "m_obj_type[]", ValueType: "id"},
	// Property type (коммерческая): m_obj_type[]=17 (участок), 18 (офис), 19 (торговое), 20 (база/склад), 22 (бизнес), 23 (своб.назначение)
	{Name: "property_type_commerce", URLParam: "m_obj_type[]", ValueType: "id"},
	// Discount filter: old_price_min=price
	{Name: "old_price", URLParam: "old_price_min", ValueType: "string"},
}

var defaultTemplates = []URLTemplate{
	{Theme: "вторичка", BasePath: "/realty/"},
	{Theme: "вторичка_студии", BasePath: "/realty/studii/"},
	{Theme: "загородка", BasePath: "/realty_out/"},
	{Theme: "загородка_дома", BasePath: "/realty_out/doma/"},
	{Theme: "загородка_участки", BasePath: "/realty_out/zemelnye-uchastki/"},
	{Theme: "новостройки", BasePath: "/zastr/"},
	{Theme: "аренда", BasePath: "/realty_rent/"},
	{Theme: "коммерческая", BasePath: "/commerce/"},
	{Theme: "ипотека", BasePath: "/ipoteka/"},
}

// --- CRUD operations ---

// UpsertFilterType creates or updates a filter type.
func (s *Store) UpsertFilterType(name, urlParam, valueType string) error {
	_, err := s.db.Exec(`
		INSERT INTO filter_types (name, url_param, value_type) VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET url_param=excluded.url_param, value_type=excluded.value_type`,
		name, urlParam, valueType)
	return err
}

// UpsertFilterValue creates or updates a filter value for a city.
func (s *Store) UpsertFilterValue(citySlug, filterType, name, urlValue, aliases string) error {
	var typeID int
	err := s.db.QueryRow("SELECT id FROM filter_types WHERE name = ?", filterType).Scan(&typeID)
	if err != nil {
		return fmt.Errorf("unknown filter type %q: %w", filterType, err)
	}
	_, err = s.db.Exec(`
		INSERT INTO filter_values (city_slug, filter_type_id, name, url_value, aliases)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(city_slug, filter_type_id, url_value) DO UPDATE
		SET name=excluded.name, aliases=excluded.aliases`,
		citySlug, typeID, name, urlValue, aliases)
	if err == nil {
		s.exportAfterWrite()
	}
	return err
}

// UpsertTemplate creates or updates a URL template.
func (s *Store) UpsertTemplate(theme, basePath string) error {
	_, err := s.db.Exec(`
		INSERT INTO url_templates (theme, base_path) VALUES (?, ?)
		ON CONFLICT(theme) DO UPDATE SET base_path=excluded.base_path`,
		theme, basePath)
	return err
}

// GetFilterValues returns all filter values for a city, optionally filtered by type.
func (s *Store) GetFilterValues(citySlug, filterType string) ([]FilterValue, error) {
	query := `
		SELECT fv.id, fv.city_slug, ft.name, fv.name, fv.url_value, fv.aliases
		FROM filter_values fv
		JOIN filter_types ft ON ft.id = fv.filter_type_id
		WHERE fv.city_slug = ?`
	args := []any{citySlug}
	if filterType != "" {
		query += " AND ft.name = ?"
		args = append(args, filterType)
	}
	query += " ORDER BY ft.name, fv.name"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FilterValue
	for rows.Next() {
		var fv FilterValue
		if err := rows.Scan(&fv.ID, &fv.CitySlug, &fv.FilterType, &fv.Name, &fv.URLValue, &fv.Aliases); err != nil {
			return nil, err
		}
		result = append(result, fv)
	}
	return result, rows.Err()
}

// GetFilterTypes returns all registered filter types.
func (s *Store) GetFilterTypes() ([]FilterType, error) {
	rows, err := s.db.Query("SELECT id, name, url_param, value_type FROM filter_types ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FilterType
	for rows.Next() {
		var ft FilterType
		if err := rows.Scan(&ft.ID, &ft.Name, &ft.URLParam, &ft.ValueType); err != nil {
			return nil, err
		}
		result = append(result, ft)
	}
	return result, rows.Err()
}

// ResolveFilterValue finds a filter value by name or alias (case-insensitive).
func (s *Store) ResolveFilterValue(citySlug, filterType, search string) (*FilterValue, error) {
	search = strings.ToLower(strings.TrimSpace(search))
	// First try exact name match.
	var fv FilterValue
	err := s.db.QueryRow(`
		SELECT fv.id, fv.city_slug, ft.name, fv.name, fv.url_value, fv.aliases
		FROM filter_values fv
		JOIN filter_types ft ON ft.id = fv.filter_type_id
		WHERE fv.city_slug = ? AND ft.name = ? AND LOWER(fv.name) = ?`,
		citySlug, filterType, search).Scan(&fv.ID, &fv.CitySlug, &fv.FilterType, &fv.Name, &fv.URLValue, &fv.Aliases)
	if err == nil {
		return &fv, nil
	}
	// Try alias match — check if search is contained in aliases.
	rows, err := s.db.Query(`
		SELECT fv.id, fv.city_slug, ft.name, fv.name, fv.url_value, fv.aliases
		FROM filter_values fv
		JOIN filter_types ft ON ft.id = fv.filter_type_id
		WHERE fv.city_slug = ? AND ft.name = ?`,
		citySlug, filterType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate FilterValue
		if err := rows.Scan(&candidate.ID, &candidate.CitySlug, &candidate.FilterType, &candidate.Name, &candidate.URLValue, &candidate.Aliases); err != nil {
			continue
		}
		for _, alias := range strings.Split(candidate.Aliases, ",") {
			if strings.TrimSpace(strings.ToLower(alias)) == search {
				return &candidate, nil
			}
		}
	}
	return nil, fmt.Errorf("filter value not found: city=%s type=%s search=%q", citySlug, filterType, search)
}

// BuildLandingURL constructs a landing URL from city, theme, and filters.
// Filter values can be comma-separated for array params (e.g. rooms="1,2").
func (s *Store) BuildLandingURL(citySlug, theme string, filters map[string]string) (string, error) {
	// Get base path from template.
	var basePath string
	err := s.db.QueryRow("SELECT base_path FROM url_templates WHERE theme = ?", theme).Scan(&basePath)
	if err != nil {
		return "", fmt.Errorf("unknown theme %q: %w", theme, err)
	}

	// Build domain.
	domain := citySlug + ".etagi.com"
	if citySlug == "tyumen" || citySlug == "тюмень" {
		domain = "www.etagi.com"
	}

	// Build query params from filters.
	params := url.Values{}
	for filterName, filterValue := range filters {
		// Get URL param name for this filter type.
		var urlParam, valueType string
		err := s.db.QueryRow("SELECT url_param, value_type FROM filter_types WHERE name = ?", filterName).Scan(&urlParam, &valueType)
		if err != nil {
			return "", fmt.Errorf("unknown filter %q: %w", filterName, err)
		}

		// Split comma-separated values for multi-value support.
		values := strings.Split(filterValue, ",")
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if valueType == "id" {
				// For ID-based filters (districts, property types), resolve name to ID.
				fv, err := s.ResolveFilterValue(citySlug, filterName, v)
				if err != nil {
					// Not found in DB — skip silently.
					continue
				}
				params.Add(urlParam, fv.URLValue)
			} else {
				params.Add(urlParam, v)
			}
		}
	}

	result := "https://" + domain + basePath
	if len(params) > 0 {
		result += "?" + params.Encode()
	}
	return result, nil
}

// BulkUpsert imports multiple filter values at once from a JSON structure.
// Expected format: [{"city_slug":"omsk","filter_type":"district","name":"Нефтяники","url_value":"42","aliases":"нефтяники,нефтяник"}]
func (s *Store) BulkUpsert(data []FilterValue) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count := 0
	for _, fv := range data {
		var typeID int
		err := tx.QueryRow("SELECT id FROM filter_types WHERE name = ?", fv.FilterType).Scan(&typeID)
		if err != nil {
			return count, fmt.Errorf("unknown filter type %q for %q: %w", fv.FilterType, fv.Name, err)
		}
		_, err = tx.Exec(`
			INSERT INTO filter_values (city_slug, filter_type_id, name, url_value, aliases)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(city_slug, filter_type_id, url_value) DO UPDATE
			SET name=excluded.name, aliases=excluded.aliases`,
			fv.CitySlug, typeID, fv.Name, fv.URLValue, fv.Aliases)
		if err != nil {
			return count, fmt.Errorf("upsert %q: %w", fv.Name, err)
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return count, err
	}
	s.exportAfterWrite()
	return count, nil
}

// Stats returns summary statistics about the database contents.
func (s *Store) Stats() (string, error) {
	type stat struct {
		label string
		query string
	}
	stats := []stat{
		{"Filter types", "SELECT COUNT(*) FROM filter_types"},
		{"URL templates", "SELECT COUNT(*) FROM url_templates"},
		{"Total filter values", "SELECT COUNT(*) FROM filter_values"},
		{"Cities with filters", "SELECT COUNT(DISTINCT city_slug) FROM filter_values"},
	}

	var sb strings.Builder
	for _, st := range stats {
		var count int
		s.db.QueryRow(st.query).Scan(&count)
		fmt.Fprintf(&sb, "%s: %d\n", st.label, count)
	}

	// Per-city breakdown.
	rows, err := s.db.Query(`
		SELECT fv.city_slug, ft.name, COUNT(*)
		FROM filter_values fv
		JOIN filter_types ft ON ft.id = fv.filter_type_id
		GROUP BY fv.city_slug, ft.name
		ORDER BY fv.city_slug, ft.name`)
	if err == nil {
		defer rows.Close()
		sb.WriteString("\nBy city:\n")
		for rows.Next() {
			var city, ftype string
			var cnt int
			rows.Scan(&city, &ftype, &cnt)
			fmt.Fprintf(&sb, "  %s / %s: %d values\n", city, ftype, cnt)
		}
	}
	return sb.String(), nil
}

// ExportJSON exports all filter values as a JSON array.
func (s *Store) ExportJSON() (string, error) {
	values, err := s.GetFilterValues("", "")
	if err != nil {
		return "", err
	}
	// Override GetFilterValues to work without city filter.
	rows, err := s.db.Query(`
		SELECT fv.city_slug, ft.name, fv.name, fv.url_value, fv.aliases
		FROM filter_values fv
		JOIN filter_types ft ON ft.id = fv.filter_type_id
		ORDER BY fv.city_slug, ft.name, fv.name`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	values = nil
	for rows.Next() {
		var fv FilterValue
		if err := rows.Scan(&fv.CitySlug, &fv.FilterType, &fv.Name, &fv.URLValue, &fv.Aliases); err != nil {
			return "", err
		}
		values = append(values, fv)
	}
	b, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ExportToFile atomically writes the current filter_values dump to exportPath.
// Safe for concurrent callers (serialized via exportMu).
// Writes to "<path>.tmp" then renames — rename is atomic inside a single
// filesystem, and on Docker bind mounts both .tmp and target share the mount.
func (s *Store) ExportToFile() error {
	if s.exportPath == "" {
		return nil
	}
	s.exportMu.Lock()
	defer s.exportMu.Unlock()

	data, err := s.ExportJSON()
	if err != nil {
		return fmt.Errorf("export json: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.exportPath), 0o755); err != nil {
		return fmt.Errorf("mkdir export: %w", err)
	}
	tmp := s.exportPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(data+"\n"), 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.exportPath); err != nil {
		// Clean up the tmp on rename failure, then bubble the error.
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// exportAfterWrite is the post-commit hook used by write methods.
// Export failures are logged but never propagated — they must not fail writes.
func (s *Store) exportAfterWrite() {
	if s.exportPath == "" {
		return
	}
	if err := s.ExportToFile(); err != nil {
		log.Printf("filters: export warning: %v", err)
	}
}

// loadSeedIfEmpty bulk-loads exportPath into filter_values if the table is empty.
// This is the git-clone recovery path: fresh DB + committed seed => working state.
// No-op if exportPath is unset, DB already has rows, or the file doesn't exist.
func (s *Store) loadSeedIfEmpty() error {
	if s.exportPath == "" {
		return nil
	}
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM filter_values").Scan(&count); err != nil {
		return fmt.Errorf("count filter_values: %w", err)
	}
	if count > 0 {
		return nil
	}
	raw, err := os.ReadFile(s.exportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read seed: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	var values []FilterValue
	if err := json.Unmarshal(raw, &values); err != nil {
		return fmt.Errorf("parse seed: %w", err)
	}
	if len(values) == 0 {
		return nil
	}
	n, err := s.BulkUpsert(values)
	if err != nil {
		return fmt.Errorf("bulk upsert seed: %w", err)
	}
	log.Printf("filters: loaded seed from %s: %d values", s.exportPath, n)
	return nil
}
