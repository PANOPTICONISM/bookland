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

	// Get original filename without extension for title fallback
	originalName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))

	var title, author, coverPath string
	switch fileType {
	case "epub":
		title, author, coverPath = extractMetadata(bookPath, bookDir)
	case "pdf":
		title, author, coverPath = extractPDFMetadata(bookPath, bookDir, originalName)
	}

	// Ensure coverPath is absolute
	if coverPath != "" && !filepath.IsAbs(coverPath) {
		coverPath = filepath.Join(DataPath, coverPath)
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

	// Set appropriate content type based on file extension
	ext := strings.ToLower(filepath.Ext(coverPath))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "image/jpeg")
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

	opfDir := filepath.Dir(opfPath)
	var coverHref string

	// Strategy 1: EPUB 3 - Look for item with properties="cover-image"
	if idx := strings.Index(opfXML, "properties=\"cover-image\""); idx != -1 {
		// Find the start of this <item> tag
		itemStart := strings.LastIndex(opfXML[:idx], "<item")
		if itemStart != -1 {
			itemEnd := strings.Index(opfXML[itemStart:], "/>")
			if itemEnd == -1 {
				itemEnd = strings.Index(opfXML[itemStart:], "</item>")
			}
			if itemEnd != -1 {
				itemTag := opfXML[itemStart : itemStart+itemEnd]
				if hrefIdx := strings.Index(itemTag, "href=\""); hrefIdx != -1 {
					start := hrefIdx + 6
					end := strings.Index(itemTag[start:], "\"")
					if end != -1 {
						coverHref = itemTag[start : start+end]
					}
				}
			}
		}
	}

	// Strategy 2: EPUB 2 - Look for meta name="cover" content="cover-id"
	if coverHref == "" {
		var coverID string
		if coverIdx := strings.Index(opfXML, "name=\"cover\""); coverIdx != -1 {
			// Find content attribute in this meta tag
			metaStart := strings.LastIndex(opfXML[:coverIdx], "<meta")
			if metaStart != -1 {
				metaEnd := strings.Index(opfXML[metaStart:], "/>")
				if metaEnd == -1 {
					metaEnd = strings.Index(opfXML[metaStart:], ">")
				}
				if metaEnd != -1 {
					metaTag := opfXML[metaStart : metaStart+metaEnd]
					if contentIdx := strings.Index(metaTag, "content=\""); contentIdx != -1 {
						start := contentIdx + 9
						end := strings.Index(metaTag[start:], "\"")
						if end != -1 {
							coverID = metaTag[start : start+end]
						}
					}
				}
			}
		}

		if coverID != "" {
			// Find item with this id that has an image media-type
			searchStr := fmt.Sprintf("id=\"%s\"", coverID)
			if itemIdx := strings.Index(opfXML, searchStr); itemIdx != -1 {
				// Find the start and end of this item tag
				itemStart := strings.LastIndex(opfXML[:itemIdx], "<item")
				if itemStart != -1 {
					itemEnd := strings.Index(opfXML[itemStart:], "/>")
					if itemEnd == -1 {
						itemEnd = strings.Index(opfXML[itemStart:], "</item>")
					}
					if itemEnd != -1 {
						itemTag := opfXML[itemStart : itemStart+itemEnd]
						// Check if this is an image
						if strings.Contains(itemTag, "media-type=\"image/") {
							if hrefIdx := strings.Index(itemTag, "href=\""); hrefIdx != -1 {
								start := hrefIdx + 6
								end := strings.Index(itemTag[start:], "\"")
								if end != -1 {
									coverHref = itemTag[start : start+end]
								}
							}
						}
					}
				}
			}
		}
	}

	// Strategy 3: Look for common cover image filenames
	if coverHref == "" {
		commonNames := []string{"cover.jpg", "cover.jpeg", "cover.png", "Cover.jpg", "Cover.jpeg", "Cover.png", "COVER.jpg", "COVER.JPG"}
		for _, f := range reader.File {
			nameLower := strings.ToLower(filepath.Base(f.Name))
			for _, cn := range commonNames {
				if strings.ToLower(cn) == nameLower {
					coverHref = f.Name
					opfDir = "" // Use absolute path from zip
					break
				}
			}
			if coverHref != "" {
				break
			}
		}
	}

	// Extract the cover image if found
	if coverHref != "" {
		var coverInZip string
		if opfDir != "" {
			coverInZip = filepath.Join(opfDir, coverHref)
		} else {
			coverInZip = coverHref
		}
		coverInZip = filepath.ToSlash(coverInZip)
		// URL decode the path (some EPUBs have encoded paths)
		coverInZip = strings.ReplaceAll(coverInZip, "%20", " ")

		for _, f := range reader.File {
			if f.Name == coverInZip || filepath.ToSlash(f.Name) == coverInZip {
				rc, err := f.Open()
				if err != nil {
					continue
				}

				// Read the file and verify it's an image
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}

				// Check magic bytes for image formats
				if !isImageFile(data) {
					log.Printf("Cover file %s is not a valid image", f.Name)
					continue
				}

				// Determine extension based on content
				ext := ".jpg"
				if len(data) > 8 && data[0] == 0x89 && data[1] == 0x50 {
					ext = ".png"
				}

				coverPath = filepath.Join(bookDir, "cover"+ext)
				outFile, err := os.Create(coverPath)
				if err != nil {
					continue
				}
				outFile.Write(data)
				outFile.Close()
				break
			}
		}
	}

	return
}

func isImageFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) > 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		if data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
			return true
		}
	}
	return false
}

func extractPDFMetadata(pdfPath, _, originalName string) (title, author, coverPath string) {
	// Use original filename as title
	title = originalName
	author = "Unknown Author"
	coverPath = ""

	// Read PDF file for metadata extraction (first 50KB should contain metadata)
	file, err := os.Open(pdfPath)
	if err != nil {
		return
	}
	defer file.Close()

	// Read first 50KB for metadata
	buffer := make([]byte, 50000)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	content := string(buffer[:n])

	// Try to extract title from PDF metadata (only if it looks valid)
	if idx := strings.Index(content, "/Title"); idx != -1 {
		if start := strings.Index(content[idx:], "("); start != -1 {
			start += idx + 1
			if end := strings.Index(content[start:], ")"); end != -1 && end < 200 {
				extractedTitle := content[start : start+end]
				extractedTitle = strings.TrimSpace(extractedTitle)
				extractedTitle = strings.ReplaceAll(extractedTitle, "\\(", "(")
				extractedTitle = strings.ReplaceAll(extractedTitle, "\\)", ")")
				// Only use extracted title if it's meaningful (not empty, not just whitespace)
				if len(extractedTitle) > 2 && len(extractedTitle) < 200 {
					title = extractedTitle
				}
			}
		}
	}

	// Try to extract author from PDF metadata
	if idx := strings.Index(content, "/Author"); idx != -1 {
		if start := strings.Index(content[idx:], "("); start != -1 {
			start += idx + 1
			if end := strings.Index(content[start:], ")"); end != -1 && end < 200 {
				extractedAuthor := content[start : start+end]
				extractedAuthor = strings.TrimSpace(extractedAuthor)
				extractedAuthor = strings.ReplaceAll(extractedAuthor, "\\(", "(")
				extractedAuthor = strings.ReplaceAll(extractedAuthor, "\\)", ")")
				if len(extractedAuthor) > 0 && len(extractedAuthor) < 200 {
					author = extractedAuthor
				}
			}
		}
	}

	// PDF cover is generated client-side using PDF.js and uploaded via UploadCover endpoint

	return
}

func UploadCover(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	// Verify book exists
	var bookDir string
	err := db.DB.QueryRow("SELECT file_path FROM books WHERE id = ?", bookID).Scan(&bookDir)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}
	bookDir = filepath.Dir(bookDir)

	err = r.ParseMultipartForm(10 << 20) // 10MB max for cover
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("cover")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Verify it's an image
	if !isImageFile(data) {
		http.Error(w, "Invalid image file", http.StatusBadRequest)
		return
	}

	// Determine extension based on content
	ext := ".jpg"
	if len(data) > 4 && data[0] == 0x89 && data[1] == 0x50 {
		ext = ".png"
	}

	coverPath := filepath.Join(bookDir, "cover"+ext)
	outFile, err := os.Create(coverPath)
	if err != nil {
		http.Error(w, "Failed to save cover", http.StatusInternalServerError)
		return
	}
	outFile.Write(data)
	outFile.Close()

	// Update database
	_, err = db.DB.Exec("UPDATE books SET cover_path = ? WHERE id = ?", coverPath, bookID)
	if err != nil {
		http.Error(w, "Failed to update book", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"coverPath": coverPath})
}
