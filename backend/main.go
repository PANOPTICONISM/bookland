package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reading/db"
	"reading/handlers"

	"github.com/gorilla/mux"
)

func main() {
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "./data"
	}

	booksPath := os.Getenv("BOOKS_PATH")
	if booksPath == "" {
		booksPath = filepath.Join(dataPath, "books")
	}

	// Convert to absolute paths
	absDataPath, err := filepath.Abs(dataPath)
	if err != nil {
		log.Fatal("Failed to get absolute data path:", err)
	}
	dataPath = absDataPath

	absBooksPath, err := filepath.Abs(booksPath)
	if err != nil {
		log.Fatal("Failed to get absolute books path:", err)
	}
	booksPath = absBooksPath

	// Create necessary directories
	err = os.MkdirAll(filepath.Join(dataPath, "covers"), 0755)
	if err != nil {
		log.Fatal("Failed to create covers directory:", err)
	}

	handlers.DataPath = dataPath

	err = db.InitDB(dataPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Scan books directory on startup
	log.Printf("Scanning books directory: %s", booksPath)
	scanBooksOnStartup(booksPath)

	r := mux.NewRouter()

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/books", handlers.GetBooks).Methods("GET")
	api.HandleFunc("/books", handlers.UploadBook).Methods("POST")
	api.HandleFunc("/books/{id}", handlers.GetBook).Methods("GET")
	api.HandleFunc("/books/{id}/file", handlers.ServeBookFile).Methods("GET")
	api.HandleFunc("/books/{id}/cover", handlers.ServeCover).Methods("GET")
	api.HandleFunc("/books/{id}/cover", handlers.UploadCover).Methods("POST")

	// Serve static frontend files in production
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath != "" {
		log.Printf("Serving static files from: %s", staticPath)
		spa := spaHandler{staticPath: staticPath, indexPath: "index.html"}
		r.PathPrefix("/").Handler(spa)
	}

	r.Use(corsMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SPA handler serves static files and falls back to index.html
type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)

	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// File does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serve the file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func scanBooksOnStartup(booksPath string) {
	addedBooks, err := handlers.ScanDirectory(booksPath)
	if err != nil {
		log.Printf("Warning: Failed to scan books directory: %v", err)
		return
	}

	if len(addedBooks) > 0 {
		log.Printf("Scan complete: Added %d books from directory", len(addedBooks))
	} else {
		log.Println("Scan complete: No new books found")
	}
}
