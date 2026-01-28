<script>
  import { onMount } from "svelte";
  import "foliate-js/view.js";
  import * as pdfjsLib from "pdfjs-dist";

  let { bookId, onClose } = $props();

  let readerContainer = $state();
  let view = $state(null);
  let loading = $state(true);
  let error = $state(null);
  let currentLocation = $state(0);
  let totalLocations = $state(0);

  let bookBlob = $state(null);
  let bookMetadata = $state(null);
  let pdfDoc = $state(null);
  let currentPage = $state(1);
  let totalPages = $state(0);

  // Header visibility
  let headerVisible = $state(true);
  let hideTimeout = null;
  let isFullscreen = $state(false);

  // Set up PDF.js worker - use worker from public directory
  pdfjsLib.GlobalWorkerOptions.workerSrc = "/pdf.worker.min.mjs";

  onMount(async () => {
    try {
      // Fetch book metadata first to determine file type
      const metadataResponse = await fetch(`/api/books/${bookId}`);
      if (!metadataResponse.ok) {
        throw new Error("Failed to load book metadata");
      }
      bookMetadata = await metadataResponse.json();

      // Fetch the book file
      const fileResponse = await fetch(`/api/books/${bookId}/file`);
      if (!fileResponse.ok) {
        throw new Error("Failed to load book");
      }
      bookBlob = await fileResponse.blob();

      loading = false;
      startHideTimer();
    } catch (err) {
      error = err.message;
      loading = false;
    }

    return () => {
      if (hideTimeout) {
        clearTimeout(hideTimeout);
      }
      if (view) {
        view.remove();
      }
      if (pdfDoc) {
        pdfDoc.destroy();
      }
    };
  });

  // Effect for EPUB rendering
  $effect(() => {
    if (
      readerContainer &&
      bookBlob &&
      bookMetadata &&
      bookMetadata.fileType === "epub" &&
      !view
    ) {
      // Create foliate-view web component
      view = document.createElement("foliate-view");
      view.style.width = "100%";
      view.style.height = "100%";
      readerContainer.appendChild(view);

      // Listen for location changes
      view.addEventListener("relocate", (e) => {
        const fraction = e.detail.fraction;
        if (fraction !== undefined) {
          currentLocation = Math.round(fraction * 100);
          totalLocations = 100;
        }
      });

      // Create a File object from the blob with proper filename
      // Foliate-js needs the filename to determine file type
      const filename = bookMetadata.title
        ? `${bookMetadata.title}.epub`
        : "book.epub";
      const file = new File([bookBlob], filename, {
        type: "application/epub+zip",
      });

      // Open the book and navigate to the start
      view
        .open(file)
        .then(() => {
          view.goTo(0);
        })
        .catch((err) => {
          error = "Failed to open EPUB: " + err.message;
        });
    }
  });

  // Effect for PDF rendering
  $effect(() => {
    if (
      readerContainer &&
      bookBlob &&
      bookMetadata &&
      bookMetadata.fileType === "pdf" &&
      !pdfDoc
    ) {
      loadPDF();
    }
  });

  async function loadPDF() {
    try {
      const arrayBuffer = await bookBlob.arrayBuffer();
      pdfDoc = await pdfjsLib.getDocument({ data: arrayBuffer }).promise;
      totalPages = pdfDoc.numPages;
      totalLocations = totalPages;
      await renderPDFPage(1);
    } catch (err) {
      error = "Failed to load PDF: " + err.message;
    }
  }

  async function renderPDFPage(pageNum) {
    if (!pdfDoc) return;

    currentPage = pageNum;
    currentLocation = pageNum;

    const page = await pdfDoc.getPage(pageNum);
    const scale = 1.5;
    const viewport = page.getViewport({ scale });

    // Clear previous content
    readerContainer.innerHTML = "";

    // Create canvas
    const canvas = document.createElement("canvas");
    const context = canvas.getContext("2d");
    canvas.height = viewport.height;
    canvas.width = viewport.width;
    canvas.style.display = "block";
    canvas.style.margin = "0 auto";

    readerContainer.appendChild(canvas);

    // Render PDF page
    await page.render({
      canvasContext: context,
      viewport: viewport,
    }).promise;
  }

  function goNext() {
    if (bookMetadata?.fileType === "epub") {
      view?.next();
    } else if (bookMetadata?.fileType === "pdf") {
      if (currentPage < totalPages) {
        renderPDFPage(currentPage + 1);
      }
    }
  }

  function goPrev() {
    if (bookMetadata?.fileType === "epub") {
      view?.prev();
    } else if (bookMetadata?.fileType === "pdf") {
      if (currentPage > 1) {
        renderPDFPage(currentPage - 1);
      }
    }
  }

  function handleKeyPress(event) {
    if (event.key === "ArrowRight") {
      goNext();
    } else if (event.key === "ArrowLeft") {
      goPrev();
    } else if (event.key === "Escape") {
      onClose();
    }
  }

  function startHideTimer() {
    if (hideTimeout) {
      clearTimeout(hideTimeout);
    }
    hideTimeout = setTimeout(() => {
      headerVisible = false;
    }, 3000);
  }

  function handleHeaderMouseEnter() {
    if (hideTimeout) {
      clearTimeout(hideTimeout);
    }
    headerVisible = true;
  }

  function handleHeaderMouseLeave() {
    startHideTimer();
  }

  function handleMouseMove(event) {
    if (event.clientY < 60 && !headerVisible) {
      headerVisible = true;
    }
  }

  function toggleFullscreen() {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen();
      isFullscreen = true;
    } else {
      document.exitFullscreen();
      isFullscreen = false;
    }
  }

  // Listen for fullscreen changes (e.g., user presses Escape)
  function handleFullscreenChange() {
    isFullscreen = !!document.fullscreenElement;
  }
</script>

<svelte:window
  onkeydown={handleKeyPress}
  onfullscreenchange={handleFullscreenChange}
/>

<div class="reader-wrapper" onmousemove={handleMouseMove} role="main">
  <div
    class="reader-header"
    class:hidden={!headerVisible}
    onmouseenter={handleHeaderMouseEnter}
    onmouseleave={handleHeaderMouseLeave}
    role="banner"
  >
    <button class="close-btn" onclick={onClose}>
      <svg
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M19 12H5M12 19l-7-7 7-7" />
      </svg>
      Back to Library
    </button>
    <button
      class="fullscreen-btn"
      onclick={toggleFullscreen}
      aria-label={isFullscreen ? "Exit fullscreen" : "Enter fullscreen"}
    >
      {#if isFullscreen}
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3"
          />
        </svg>
      {:else}
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"
          />
        </svg>
      {/if}
    </button>
  </div>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>Loading book...</p>
    </div>
  {:else if error}
    <div class="error">
      <p>Error: {error}</p>
      <button onclick={onClose}>Go Back</button>
    </div>
  {:else}
    <div class="reader-container" bind:this={readerContainer}></div>

    {#if totalLocations > 0}
      <div class="progress-bar">
        <span>{Math.round((currentLocation / totalLocations) * 100)}%</span>
      </div>
    {/if}
  {/if}
</div>

<style>
  :root {
    --background-color: #f5f1e8;
  }

  .reader-wrapper {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: var(--background-color);
    display: flex;
    flex-direction: column;
    z-index: 1000;
  }

  .reader-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 2rem;
    background: var(--background-color);
    border-bottom: 1px solid #e2e8f0;
    z-index: 1001;
    transform: translateY(0);
    transition: transform 0.3s ease-in-out;
  }

  .reader-header.hidden {
    transform: translateY(-100%);
  }

  .close-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1rem;
    color: #4a5568;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    transition: background 0.2s;
  }

  .close-btn:hover {
    background: #f7fafc;
  }

  .fullscreen-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 6px;
    color: #4a5568;
    transition: background 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .fullscreen-btn:hover {
    background: #f7fafc;
  }

  .reader-container {
    flex: 1;
    overflow: auto;
    position: relative;
    max-width: 900px;
    margin: 0 auto;
    width: 100%;
    padding: 2rem;
    padding-top: 4rem;
  }

  .loading,
  .error {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    color: #4a5568;
    padding-top: 4rem;
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

  .error button {
    padding: 0.75rem 1.5rem;
    background: #4299e1;
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1rem;
  }

  .progress-bar {
    position: fixed;
    bottom: 1.5rem;
    right: 1.5rem;
    font-size: 0.85rem;
    color: #718096;
    z-index: 1001;
  }

  @media (max-width: 768px) {
    .reader-container {
      padding: 1rem;
      padding-top: 4rem;
    }
  }
</style>
