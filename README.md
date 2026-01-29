# Bookland - A performant, low power library for your books

Made with Go and Svelte.

## Features

- Upload and manage EPUB and PDF files
- Fast, responsive reader with Foliate-js (EPUB) and PDF.js (PDF)
- Automatic metadata extraction (title, author, cover)
- Auto-scan books from a mounted directory on startup
- Clean, minimal UI
- Single Docker container deployment
- Persistent storage with volumes

## Tech Stack

**Backend:**
- Go (fast, efficient)
- SQLite (metadata storage)
- Gorilla Mux (routing)

**Frontend:**
- Svelte (reactive, compiled)
- Foliate-js (EPUB rendering)
- PDF.js (PDF rendering)
- Vite (build tool)

## Self-Hosting with Docker

1. Clone this repository:
   ```bash
   git clone <repo-url>
   cd bookland
   ```

2. Start the application:
   ```bash
   docker-compose up -d
   ```

3. Open your browser:
   ```
   http://localhost:8080
   ```

That's it! Your books are stored in a Docker volume and persist between restarts.

## Usage

1. **Upload Books**: Drag and drop EPUB or PDF files, or click the upload area
2. **Auto-import**: Place books in `BOOKS_PATH` and they'll be scanned on startup
3. **View Library**: See all your books with covers in a grid
4. **Read**: Click any book to open the reader
5. **Navigate**: Use arrow keys or swipe to turn pages

## File Structure

```
bookland/
├── backend/              # Go API server
│   ├── main.go          # Entry point
│   ├── handlers/        # HTTP handlers
│   ├── models/          # Data models
│   └── db/              # Database setup
├── frontend/            # Svelte app
│   └── src/
│       ├── App.svelte
│       └── components/
│           ├── Library.svelte
│           └── Reader.svelte
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Development

**Backend:**
```bash
cd backend
DATA_PATH=./data BOOKS_PATH=~/Downloads go run .
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

The frontend dev server proxies `/api` requests to `localhost:8080`.

## Configuration

**Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `DATA_PATH` | Where database and covers are stored | `./data` |
| `BOOKS_PATH` | Where to scan for book files (can be read-only) | `DATA_PATH/books` |
| `PORT` | Server port | `8080` |
| `STATIC_PATH` | Path to built frontend (production only) | - |

## Storage

- **DATA_PATH**: Contains the SQLite database and extracted covers. Must be writable.
- **BOOKS_PATH**: Source directory for book files. Scanned on startup. Can be read-only.

In Docker, `DATA_PATH` uses a named volume (`book-data`) while `BOOKS_PATH` can be mounted from your host (e.g., `/home/user/books`).

## Performance

- **Small Docker image**: ~30MB Alpine-based
- **Fast uploads**: Go handles EPUB processing efficiently
- **Smooth reading**: Foliate-js provides native-like pagination
- **Low memory**: Optimized for self-hosting on modest hardware (Raspberry Pi compatible)

## License

MIT
