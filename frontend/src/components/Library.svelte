<script>
  import { onMount } from "svelte";

  export let onOpenBook;

  let books = [];
  let uploading = false;
  let dragOver = false;

  onMount(async () => {
    await fetchBooks();
  });

  async function fetchBooks() {
    try {
      const response = await fetch("/api/books");
      const data = await response.json();
      books = Array.isArray(data) ? data : [];
    } catch (error) {
      console.error("Failed to fetch books:", error);
      books = [];
    }
  }

  async function handleFileSelect(event) {
    const files = event.target.files || event.dataTransfer?.files;
    if (!files || files.length === 0) {
      return;
    };

    const file = files[0];
    const filename = file.name.toLowerCase();
    if (!filename.endsWith(".epub") && !filename.endsWith(".pdf")) {
      alert("Please select an EPUB or PDF file");
      return;
    }

    await uploadBook(file);
  }

  async function uploadBook(file) {
    uploading = true;
    const formData = new FormData();
    formData.append("book", file);

    try {
      const response = await fetch("/api/books", {
        method: "POST",
        body: formData,
      });

      if (response.ok) {
        await fetchBooks();
      } else {
        alert("Failed to upload book");
      }
    } catch (error) {
      console.error("Upload error:", error);
      alert("Failed to upload book");
    } finally {
      uploading = false;
    }
  }

  function handleDragOver(event) {
    event.preventDefault();
    dragOver = true;
  }

  function handleDragLeave() {
    dragOver = false;
  }

  function handleDrop(event) {
    event.preventDefault();
    dragOver = false;
    handleFileSelect(event);
  }
</script>

<div class="container">
  <header>
    <h1>My Library</h1>
  </header>
  <div
    class="upload-zone"
    class:drag-over={dragOver}
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
    role="button"
    tabindex="0"
  >
    <input
      type="file"
      accept=".epub,.pdf"
      onchange={handleFileSelect}
      id="file-input"
      style="display: none;"
    />
    <label for="file-input">
      {#if uploading}
        <div class="spinner"></div>
        <p>Uploading...</p>
      {:else}
        <svg
          width="48"
          height="48"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="17 8 12 3 7 8" />
          <line x1="12" y1="3" x2="12" y2="15" />
        </svg>
        <p>Drop EPUB or PDF file here or click to upload</p>
      {/if}
    </label>
  </div>
  {#if books.length > 0}
    <div class="books-grid">
      {#each books as book (book.id)}
        <button
          type="button"
          class="book-card"
          onclick={() => onOpenBook(book.id)}>
          <div class="cover-container">
            {#if book.coverPath}
              <img src="/api/books/{book.id}/cover" alt={book.title} />
            {:else}
              <div class="no-cover">
                <svg
                  width="48"
                  height="48"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                >
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
                  <path
                    d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"
                  />
                </svg>
              </div>
            {/if}
            <span class="file-type-tag" class:pdf={book.fileType === "pdf"}>
              {book.fileType?.toUpperCase() || "EPUB"}
            </span>
          </div>
          <div class="book-info">
            <h3>{book.title}</h3>
            <p>{book.author}</p>
          </div>
        </button>
      {/each}
    </div>
  {:else}
    <div class="empty-state">
      <svg
        width="64"
        height="64"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
      >
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
        <path
          d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"
        />
      </svg>
      <p>No books yet. Upload your first EPUB or PDF!</p>
    </div>
  {/if}
</div>

<style>
  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
  }

  header {
    margin-bottom: 2rem;
  }

  h1 {
    font-size: 2rem;
    font-weight: 600;
    color: #1a1a1a;
  }

  .upload-zone {
    border: 2px dashed #cbd5e0;
    border-radius: 12px;
    padding: 3rem;
    text-align: center;
    margin-bottom: 3rem;
    background: white;
    transition: all 0.2s;
    cursor: pointer;
  }

  .upload-zone:hover,
  .upload-zone.drag-over {
    border-color: #4299e1;
    background: #ebf8ff;
  }

  .upload-zone label {
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    color: #4a5568;
  }

  .spinner {
    width: 48px;
    height: 48px;
    border: 4px solid #e2e8f0;
    border-top-color: #4299e1;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .books-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 1.5rem;
  }

  .book-card {
    cursor: pointer;
    border-radius: 8px;
    overflow: hidden;
    background: white;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    transition:
      transform 0.2s,
      box-shadow 0.2s;
    border: none;
    padding: 0;
  }

  .book-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  }

  .cover-container {
    position: relative;
  }

  .book-card img {
    width: 100%;
    height: 240px;
    object-fit: cover;
    display: block;
  }

  .file-type-tag {
    position: absolute;
    top: 8px;
    right: 8px;
    background: rgba(102, 126, 234, 0.9);
    color: white;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 3px 6px;
    border-radius: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .file-type-tag.pdf {
    background: rgba(229, 62, 62, 0.9);
  }

  .no-cover {
    width: 100%;
    height: 240px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }

  .book-info {
    padding: 1rem;
  }

  .book-info h3 {
    font-size: 0.95rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
    color: #1a1a1a;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .book-info p {
    font-size: 0.85rem;
    color: #718096;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-state {
    text-align: center;
    padding: 2rem;
    color: #a0aec0;
  }

  .empty-state svg {
    margin-bottom: 1rem;
  }

  .empty-state p {
    font-size: 1.1rem;
  }
</style>
