package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reading/db"
	"reading/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var DataPath string

func UploadBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("book")
	if err != nil {
		// Try legacy field name for backwards compatibility
		file, header, err = r.FormFile("epub")
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
	}
	defer file.Close()

	filename := strings.ToLower(header.Filename)
	var fileType string
	var fileExt string

	if strings.HasSuffix(filename, ".epub") {
		fileType = "epub"
		fileExt = ".epub"
	} else if strings.HasSuffix(filename, ".pdf") {
		fileType = "pdf"
		fileExt = ".pdf"
	} else {
		http.Error(w, "Only .epub and .pdf files are supported", http.StatusBadRequest)
		return
	}

	bookID := uuid.New().String()
	bookDir := filepath.Join(DataPath, "books", bookID)
	err = os.MkdirAll(bookDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create book directory", http.StatusInternalServerError)
		return
	}

	bookPath := filepath.Join(bookDir, "book"+fileExt)
	dst, err := os.Create(bookPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	fileSize, err := io.Copy(dst, file)
	dst.Close()
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	var title, author, coverPath string
	switch fileType {
	case "epub":
		title, author, coverPath = extractMetadata(bookPath, bookDir)
	case "pdf":
		title, author, coverPath = extractPDFMetadata(bookPath, bookDir)
	}

	book := models.Book{
		ID:        bookID,
		Title:     title,
		Author:    author,
		CoverPath: coverPath,
		FilePath:  bookPath,
		FileSize:  fileSize,
		FileType:  fileType,
		AddedAt:   time.Now(),
	}

	_, err = db.DB.Exec(
		"INSERT INTO books (id, title, author, cover_path, file_path, file_size, file_type, added_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		book.ID, book.Title, book.Author, book.CoverPath, book.FilePath, book.FileSize, book.FileType, book.AddedAt,
	)
	if err != nil {
		http.Error(w, "Failed to save book metadata", http.StatusInternalServerError)
		log.Println("DB error:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func GetBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, title, author, cover_path, file_path, file_size, file_type, added_at FROM books ORDER BY added_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch books", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := make([]models.Book, 0)
	for rows.Next() {
		var book models.Book
		err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.CoverPath, &book.FilePath, &book.FileSize, &book.FileType, &book.AddedAt)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		books = append(books, book)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func GetBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	var book models.Book
	err := db.DB.QueryRow(
		"SELECT id, title, author, cover_path, file_path, file_size, file_type, added_at FROM books WHERE id = ?",
		bookID,
	).Scan(&book.ID, &book.Title, &book.Author, &book.CoverPath, &book.FilePath, &book.FileSize, &book.FileType, &book.AddedAt)

	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func ServeBookFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	var filePath, fileType string
	err := db.DB.QueryRow("SELECT file_path, file_type FROM books WHERE id = ?", bookID).Scan(&filePath, &fileType)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	// Set appropriate content type based on file type
	if fileType == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
	} else {
		w.Header().Set("Content-Type", "application/epub+zip")
	}

	http.ServeFile(w, r, filePath)
}

func ServeCover(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	var coverPath string
	err := db.DB.QueryRow("SELECT cover_path FROM books WHERE id = ?", bookID).Scan(&coverPath)
	if err != nil || coverPath == "" {
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, coverPath)
}

func extractMetadata(epubPath, bookDir string) (title, author, coverPath string) {
	title = "Unknown Title"
	author = "Unknown Author"
	coverPath = ""

	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return
	}
	defer reader.Close()

	var containerXML string
	var opfPath string

	for _, f := range reader.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			containerXML = string(data)
			break
		}
	}

	if idx := strings.Index(containerXML, "full-path=\""); idx != -1 {
		start := idx + 11
		end := strings.Index(containerXML[start:], "\"")
		if end != -1 {
			opfPath = containerXML[start : start+end]
		}
	}

	if opfPath == "" {
		return
	}

	var opfXML string
	for _, f := range reader.File {
		if f.Name == opfPath {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			opfXML = string(data)
			break
		}
	}

	if titleIdx := strings.Index(opfXML, "<dc:title>"); titleIdx != -1 {
		start := titleIdx + 10
		end := strings.Index(opfXML[start:], "</dc:title>")
		if end != -1 {
			title = opfXML[start : start+end]
		}
	}

	if authorIdx := strings.Index(opfXML, "<dc:creator"); authorIdx != -1 {
		contentStart := strings.Index(opfXML[authorIdx:], ">")
		if contentStart != -1 {
			start := authorIdx + contentStart + 1
			end := strings.Index(opfXML[start:], "</dc:creator>")
			if end != -1 {
				author = opfXML[start : start+end]
			}
		}
	}

	var coverID string
	if coverIdx := strings.Index(opfXML, "name=\"cover\""); coverIdx != -1 {
		contentStart := strings.LastIndex(opfXML[:coverIdx], "content=\"")
		if contentStart != -1 {
			start := contentStart + 9
			end := strings.Index(opfXML[start:], "\"")
			if end != -1 {
				coverID = opfXML[start : start+end]
			}
		}
	}

	if coverID != "" {
		searchStr := fmt.Sprintf("id=\"%s\"", coverID)
		if itemIdx := strings.Index(opfXML, searchStr); itemIdx != -1 {
			hrefStart := strings.Index(opfXML[itemIdx:], "href=\"")
			if hrefStart != -1 {
				start := itemIdx + hrefStart + 6
				end := strings.Index(opfXML[start:], "\"")
				if end != -1 {
					coverHref := opfXML[start : start+end]
					opfDir := filepath.Dir(opfPath)
					coverInZip := filepath.Join(opfDir, coverHref)
					coverInZip = filepath.ToSlash(coverInZip)

					for _, f := range reader.File {
						if f.Name == coverInZip {
							rc, err := f.Open()
							if err != nil {
								continue
							}
							coverPath = filepath.Join(bookDir, "cover.jpg")
							outFile, err := os.Create(coverPath)
							if err != nil {
								rc.Close()
								continue
							}
							io.Copy(outFile, rc)
							outFile.Close()
							rc.Close()
							break
						}
					}
				}
			}
		}
	}

	return
}

func extractPDFMetadata(pdfPath, bookDir string) (title, author, coverPath string) {
	title = "Unknown Title"
	author = "Unknown Author"
	coverPath = ""

	// Read PDF file for basic metadata extraction
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		return
	}

	content := string(data)

	// Try to extract title from PDF metadata
	if idx := strings.Index(content, "/Title"); idx != -1 {
		start := strings.Index(content[idx:], "(")
		if start != -1 {
			start += idx + 1
			end := strings.Index(content[start:], ")")
			if end != -1 && end < 200 {
				extractedTitle := content[start : start+end]
				extractedTitle = strings.TrimSpace(extractedTitle)
				if len(extractedTitle) > 0 && len(extractedTitle) < 200 {
					title = extractedTitle
				}
			}
		}
	}

	// Try to extract author from PDF metadata
	if idx := strings.Index(content, "/Author"); idx != -1 {
		start := strings.Index(content[idx:], "(")
		if start != -1 {
			start += idx + 1
			end := strings.Index(content[start:], ")")
			if end != -1 && end < 200 {
				extractedAuthor := content[start : start+end]
				extractedAuthor = strings.TrimSpace(extractedAuthor)
				if len(extractedAuthor) > 0 && len(extractedAuthor) < 200 {
					author = extractedAuthor
				}
			}
		}
	}

	// PDF cover generation would require external tools (like ImageMagick)
	// For now, we'll skip cover generation for PDFs
	// TODO: Implement PDF thumbnail generation

	return
}
