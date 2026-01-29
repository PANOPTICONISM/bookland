package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reading/db"
	"reading/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ScanDirectory scans a directory for book files and adds them to the database.
// Returns the list of added books and any error encountered.
func ScanDirectory(booksDir string) ([]models.Book, error) {
	entries, err := os.ReadDir(booksDir)
	if err != nil {
		return nil, err
	}

	var addedBooks []models.Book
	coversDir := filepath.Join(DataPath, "covers")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		filePath := filepath.Join(booksDir, filename)

		// Check if it's an EPUB or PDF
		lowerName := strings.ToLower(filename)
		isEpub := strings.HasSuffix(lowerName, ".epub")
		isPdf := strings.HasSuffix(lowerName, ".pdf")

		if !isEpub && !isPdf {
			continue
		}

		// Check if book already exists in database by file path
		var existingID string
		err := db.DB.QueryRow(
			"SELECT id FROM books WHERE file_path = ?",
			filePath,
		).Scan(&existingID)

		if err == nil {
			// Book already exists
			continue
		}

		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Failed to stat file %s: %v", filename, err)
			continue
		}

		// Generate unique ID for the book
		bookID := uuid.New().String()

		// Extract metadata using shared functions
		var title, author, coverPath string
		fileType := "pdf"

		if isEpub {
			fileType = "epub"
			title, author, coverPath = ExtractEPUBMetadata(filePath, coversDir)

			// Rename cover file to use bookID instead of generic "cover"
			if coverPath != "" {
				ext := filepath.Ext(coverPath)
				newCoverPath := filepath.Join(coversDir, bookID+ext)
				if err := os.Rename(coverPath, newCoverPath); err != nil {
					log.Printf("Failed to rename cover: %v", err)
				} else {
					coverPath = newCoverPath
				}
			}
		} else {
			originalName := strings.TrimSuffix(filename, filepath.Ext(filename))
			title, author = ExtractPDFMetadata(filePath, originalName)
			coverPath = ExtractPDFCover(filePath, coversDir, bookID)
		}

		book := models.Book{
			ID:        bookID,
			Title:     title,
			Author:    author,
			CoverPath: coverPath,
			FilePath:  filePath,
			FileSize:  info.Size(),
			FileType:  fileType,
			AddedAt:   time.Now(),
		}

		// Insert into database
		_, err = db.DB.Exec(
			"INSERT INTO books (id, title, author, cover_path, file_path, file_size, file_type, added_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			book.ID,
			book.Title,
			book.Author,
			book.CoverPath,
			book.FilePath,
			book.FileSize,
			book.FileType,
			book.AddedAt,
		)

		if err != nil {
			log.Printf("Failed to insert book %s: %v", title, err)
			continue
		}

		addedBooks = append(addedBooks, book)
		log.Printf("Added book: %s by %s", title, author)
	}

	return addedBooks, nil
}

// ScanBooksDirectory is an HTTP handler that scans the books directory
func ScanBooksDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	booksDir := filepath.Join(DataPath, "books")

	// Create books directory if it doesn't exist
	if err := os.MkdirAll(booksDir, 0755); err != nil {
		http.Error(w, "Failed to create books directory", http.StatusInternalServerError)
		return
	}

	addedBooks, err := ScanDirectory(booksDir)
	if err != nil {
		http.Error(w, "Failed to scan directory", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Scan completed",
		"added":   len(addedBooks),
		"books":   addedBooks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
