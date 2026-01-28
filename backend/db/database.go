package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dataPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataPath+"/books.db")
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
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_added_at ON books(added_at DESC);
	`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}
