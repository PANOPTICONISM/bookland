# Reading - Self-Hosted EPUB Reader

A performant, self-hosted web-based EPUB reader built with Go and Svelte.

## Features

- Upload and manage EPUB files
- Fast, responsive reader with Foliate-js
- Automatic metadata extraction (title, author, cover)
- Clean, minimal UI
- Single Docker container deployment
- Persistent storage with volumes

## Tech Stack

**Backend:**
- Go 1.22 (fast, efficient)
- SQLite (metadata storage)
- Gorilla Mux (routing)

**Frontend:**
- Svelte (reactive, compiled)
- Foliate-js (EPUB rendering)
- Vite (build tool)

## Self-Hosting with Docker

1. Clone this repository:
   ```bash
   git clone <repo-url>
   cd reading
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

To stop:
```bash
docker-compose down
```

To view logs:
```bash
docker-compose logs -f
```

## Development

**Terminal 1 - Backend:**
```bash
cd backend
go mod download
DATA_PATH=./data go run main.go
```

**Terminal 2 - Frontend:**
```bash
cd frontend
npm install
npm run dev
```

The frontend dev server runs on `http://localhost:5173` and proxies API calls to the backend on port 8080.

## Usage

1. **Upload Books**: Drag and drop EPUB files or click the upload area
2. **View Library**: See all your books with covers in a grid
3. **Read**: Click any book to open the reader
4. **Navigate**: Use arrow keys or on-screen buttons to turn pages

## File Structure

```
reading/
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

## Configuration

**Environment Variables:**

- `DATA_PATH`: Where books and database are stored (default: `./data`)
- `PORT`: Server port (default: `8080`)

**Changing the port:**

Edit `docker-compose.yml`:
```yaml
ports:
  - "3000:8080"  # Access on port 3000
```

## Storage

Books are stored in the `book-data` Docker volume. The volume persists even if you remove the container.

To backup your books:
```bash
docker run --rm -v reading_book-data:/data -v $(pwd):/backup alpine tar czf /backup/books-backup.tar.gz /data
```

To restore from backup:
```bash
docker run --rm -v reading_book-data:/data -v $(pwd):/backup alpine tar xzf /backup/books-backup.tar.gz -C /
```

## Performance

- **Small Docker image**: ~30MB Alpine-based
- **Fast uploads**: Go handles EPUB processing efficiently
- **Smooth reading**: Foliate-js provides native-like pagination
- **Low memory**: Optimized for self-hosting on modest hardware (Raspberry Pi compatible)

## License

MIT
