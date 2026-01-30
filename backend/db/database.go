package db

import (
	"database/sql"
	"log"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dataPath string) error {
	var err error
	// Enable WAL mode and set busy timeout for better concurrency
	DB, err = sql.Open("sqlite", dataPath+"/books.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}

	// Limit connections to avoid lock contention
	DB.SetMaxOpenConns(1)

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		cover_path TEXT,
		file_path TEXT NOT NULL,
		file_size INTEGER,
		file_type TEXT DEFAULT 'epub',
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_added_at ON books(added_at DESC);
	`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// Migration: Add file_type column if it doesn't exist
	_, err = DB.Exec(`ALTER TABLE books ADD COLUMN file_type TEXT DEFAULT 'epub'`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	// Migration: Add reading_progress column if it doesn't exist
	_, err = DB.Exec(`ALTER TABLE books ADD COLUMN reading_progress TEXT`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}
