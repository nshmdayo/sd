package history

import (
	"cmp"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection for history management.
type DB struct {
	conn *sql.DB
}

// Entry represents one row in the history table.
type Entry struct {
	ID         int64
	Path       string
	VisitCount int
	LastVisit  time.Time
	CreatedAt  time.Time
	Frecency   float64 // computed, not stored
}

// SortOrder controls how List results are ordered.
type SortOrder int

const (
	SortFrecency SortOrder = iota
	SortTime
	SortAlpha
)

// Open opens (or creates) the SQLite history database at dbPath.
func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			path        TEXT    NOT NULL UNIQUE,
			visit_count INTEGER NOT NULL DEFAULT 1,
			last_visit  TEXT    NOT NULL,
			created_at  TEXT    NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_last_visit ON history(last_visit);
	`)
	return err
}

// Record upserts a path into the history table.
func (db *DB) Record(path string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(`
		INSERT INTO history (path, visit_count, last_visit, created_at)
		VALUES (?, 1, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			visit_count = visit_count + 1,
			last_visit  = excluded.last_visit
	`, path, now, now)
	return err
}

// List returns history entries ordered by the given sort order.
func (db *DB) List(sort SortOrder, limit int) ([]Entry, error) {
	var orderBy string
	switch sort {
	case SortTime:
		orderBy = "last_visit DESC"
	case SortAlpha:
		orderBy = "path ASC"
	default:
		orderBy = "last_visit DESC" // frecency computed in Go
	}

	rows, err := db.conn.Query(
		`SELECT id, path, visit_count, last_visit, created_at FROM history ORDER BY `+orderBy+` LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var lastVisit, createdAt string
		if err := rows.Scan(&e.ID, &e.Path, &e.VisitCount, &lastVisit, &createdAt); err != nil {
			return nil, err
		}
		e.LastVisit, _ = time.Parse(time.RFC3339, lastVisit)
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		e.Frecency = frecencyScore(e.VisitCount, e.LastVisit)
		entries = append(entries, e)
	}

	if sort == SortFrecency {
		slices.SortFunc(entries, func(a, b Entry) int {
			return cmp.Compare(b.Frecency, a.Frecency) // descending
		})
	}
	return entries, rows.Err()
}

// GetByIndex returns the N-th entry (1-based) ordered by frecency.
func (db *DB) GetByIndex(n int) (*Entry, error) {
	entries, err := db.List(SortFrecency, n)
	if err != nil {
		return nil, err
	}
	if n < 1 || n > len(entries) {
		return nil, fmt.Errorf("history index %d out of range (have %d entries)", n, len(entries))
	}
	e := entries[n-1]
	return &e, nil
}

// Clear deletes all history.
func (db *DB) Clear() error {
	_, err := db.conn.Exec(`DELETE FROM history`)
	return err
}

// Prune removes oldest entries exceeding maxEntries.
func (db *DB) Prune(maxEntries int) error {
	_, err := db.conn.Exec(`
		DELETE FROM history WHERE id NOT IN (
			SELECT id FROM history ORDER BY last_visit DESC LIMIT ?
		)
	`, maxEntries)
	return err
}

// FrecencyMap returns a map[path]frecencyScore for all entries.
// Used by fuzzy search to boost directory scores.
func (db *DB) FrecencyMap() (map[string]float64, error) {
	rows, err := db.conn.Query(`SELECT path, visit_count, last_visit FROM history`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]float64)
	for rows.Next() {
		var path, lastVisit string
		var count int
		if err := rows.Scan(&path, &count, &lastVisit); err != nil {
			return nil, err
		}
		lv, _ := time.Parse(time.RFC3339, lastVisit)
		m[path] = frecencyScore(count, lv)
	}
	return m, rows.Err()
}

func frecencyScore(visitCount int, lastVisit time.Time) float64 {
	hours := time.Since(lastVisit).Hours()
	var weight float64
	switch {
	case hours < 1:
		weight = 4.0
	case hours < 24:
		weight = 2.0
	case hours < 24*7:
		weight = 1.0
	case hours < 24*30:
		weight = 0.5
	default:
		weight = 0.25
	}
	return float64(visitCount) * weight
}

