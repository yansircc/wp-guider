package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func openDB() *sql.DB {
	os.MkdirAll(trainingDir, 0o755)
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		fatal("cannot open database: " + err.Error())
	}
	initDB(db)
	return db
}

func initDB(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attempts (
			id INTEGER PRIMARY KEY,
			topic TEXT NOT NULL,
			task_id TEXT NOT NULL,
			difficulty INTEGER,
			passed INTEGER NOT NULL,
			error_detail TEXT,
			duration_sec INTEGER,
			created_at TEXT DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS topic_mastery (
			topic TEXT PRIMARY KEY,
			consecutive_passes INTEGER DEFAULT 0,
			mastered INTEGER DEFAULT 0,
			mastered_at TEXT,
			next_review TEXT,
			updated_at TEXT DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY,
			started_at TEXT DEFAULT (datetime('now')),
			ended_at TEXT
		);
		CREATE TABLE IF NOT EXISTS current_task (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			task_json TEXT NOT NULL,
			issued_at TEXT DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		fatal("cannot init database: " + err.Error())
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// scanMap scans a single row into a map.
func scanMap(row *sql.Row, cols ...string) (map[string]any, error) {
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		return nil, err
	}
	m := make(map[string]any, len(cols))
	for i, c := range cols {
		m[c] = vals[i]
	}
	return m, nil
}

func dbGetInt(db *sql.DB, query string, args ...any) int {
	var n int
	err := db.QueryRow(query, args...).Scan(&n)
	if err != nil {
		return 0
	}
	return n
}

func log(msg string) {
	fmt.Fprintln(os.Stderr, "::", msg)
}
