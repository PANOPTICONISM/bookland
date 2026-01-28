# Bookland - A performant, low power library for your books

Made with Go and Svelte.

## Features

- Upload and manage EPUB files
- Fast, responsive reader with Foliate-js
- Automatic metadata extraction (title, author, cover)
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

1. **Upload Books**: Drag and drop EPUB files or click the upload area
2. **View Library**: See all your books with covers in a grid
3. **Read**: Click any book to open the reader and keep track of your progress
4. **Navigate**: Use arrow keys or screen touch to turn pages

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

## Configuration

**Environment Variables:**

- `DATA_PATH`: Where books and database are stored (default: `./data`)
- `PORT`: Server port (default: `8080`)

## Storage

Books are stored in the `book-data` Docker volume.

## Performance

- **Small Docker image**: ~30MB Alpine-based
- **Fast uploads**: Go handles EPUB processing efficiently
- **Smooth reading**: Foliate-js provides native-like pagination
- **Low memory**: Optimized for self-hosting on modest hardware (Raspberry Pi compatible)

## License

MIT
