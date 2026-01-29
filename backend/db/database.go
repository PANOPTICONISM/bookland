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
	DB, err = sql.Open("sqlite", dataPath+"/books.db")
	if err != nil {
		return err
	}

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
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}
