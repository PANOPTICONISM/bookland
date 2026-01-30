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

  // Detect touch-only device (no mouse)
  const isTouchDevice = typeof window !== 'undefined' && window.matchMedia('(pointer: coarse)').matches;

  // Set up PDF.js worker - use worker from public directory
  pdfjsLib.GlobalWorkerOptions.workerSrc = "/pdf.worker.min.mjs";

  // Save reading progress to backend (debounced)
  let saveTimeout = null;
  function saveProgress(progress) {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }
    saveTimeout = setTimeout(async () => {
      try {
        await fetch(`/api/books/${bookId}/progress`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ progress })
        });
      } catch (err) {
        // Silently fail - don't interrupt reading
      }
    }, 1000); // Wait 1 second after last change before saving
  }

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
      if (saveTimeout) {
        clearTimeout(saveTimeout);
      }
      if (view) {
        try {
          view.remove();
        } catch (e) {
          // Ignore errors during cleanup
        }
        view = null;
      }
      if (pdfDoc) {
        pdfDoc.destroy();
        pdfDoc = null;
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
        // Save progress with CFI for precise location
        const cfi = e.detail.cfi;
        if (cfi) {
          saveProgress(JSON.stringify({ type: 'epub', cfi, fraction }));
        }
      });

      view.addEventListener("load", (e) => {
        try {
          const doc = e.detail?.doc;
          if (doc && doc.documentElement && doc.head) {
            const style = doc.createElement("style");
            style.textContent = `
              p {
                font-size: 20px;
              }
            `;
            doc.head.appendChild(style);
          }
        } catch (err) {
          // Ignore styling errors
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

      // Open the book and restore saved progress or start from beginning
      view
        .open(file)
        .then(() => {
          if (bookMetadata.readingProgress) {
            try {
              const progress = JSON.parse(bookMetadata.readingProgress);
              if (progress.type === 'epub' && progress.cfi) {
                view.goTo(progress.cfi);
                return;
              }
            } catch (e) {
              // Invalid progress data, start from beginning
            }
          }
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

      // Restore saved progress or start from page 1
      let startPage = 1;
      if (bookMetadata.readingProgress) {
        try {
          const progress = JSON.parse(bookMetadata.readingProgress);
          if (progress.type === 'pdf' && progress.page) {
            startPage = Math.min(progress.page, totalPages);
          }
        } catch (e) {
          // Invalid progress data
        }
      }
      await renderPDFPage(startPage);
    } catch (err) {
      error = "Failed to load PDF: " + err.message;
    }
  }

  async function renderPDFPage(pageNum) {
    if (!pdfDoc || !readerContainer) return;

    currentPage = pageNum;
    currentLocation = pageNum;

    const page = await pdfDoc.getPage(pageNum);

    const containerHeight = window.innerHeight - 80; // Account for header and padding
    const containerWidth = Math.min(window.innerWidth - 40, 860); // Max width with padding
    const originalViewport = page.getViewport({ scale: 1 });
    const scaleHeight = containerHeight / originalViewport.height;
    const scaleWidth = containerWidth / originalViewport.width;
    const scale = Math.min(scaleHeight, scaleWidth);
    const viewport = page.getViewport({ scale });

    // Clear previous content
    readerContainer.innerHTML = "";

    // Create wrapper for canvas and text layer
    const wrapper = document.createElement("div");
    wrapper.style.position = "relative";
    wrapper.style.width = `${viewport.width}px`;
    wrapper.style.height = `${viewport.height}px`;
    wrapper.style.margin = "0 auto";

    // Create canvas
    const canvas = document.createElement("canvas");
    const context = canvas.getContext("2d");
    canvas.height = viewport.height;
    canvas.width = viewport.width;
    canvas.style.display = "block";

    wrapper.appendChild(canvas);

    // Create text layer for selection
    const textLayerDiv = document.createElement("div");
    textLayerDiv.className = "pdf-text-layer";
    textLayerDiv.style.position = "absolute";
    textLayerDiv.style.left = "0";
    textLayerDiv.style.top = "0";
    textLayerDiv.style.right = "0";
    textLayerDiv.style.bottom = "0";
    textLayerDiv.style.overflow = "hidden";
    textLayerDiv.style.lineHeight = "1";
    wrapper.appendChild(textLayerDiv);

    readerContainer.appendChild(wrapper);

    // Render PDF page
    await page.render({
      canvasContext: context,
      viewport: viewport,
    }).promise;

    // Render text layer for selection
    const textContent = await page.getTextContent();
    const textLayer = new pdfjsLib.TextLayer({
      textContentSource: textContent,
      container: textLayerDiv,
      viewport: viewport,
    });
    await textLayer.render();

    saveProgress(JSON.stringify({ type: 'pdf', page: pageNum, totalPages }));
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

  let touchStartX = 0;
  let touchStartY = 0;

  function handleKeyPress(event) {
    if (event.key === "ArrowRight") {
      goNext();
    } else if (event.key === "ArrowLeft") {
      goPrev();
    } else if (event.key === "Escape") {
      onClose();
    }
  }

  // All touch handling on wrapper for full-screen coverage
  function handleTouchStart(event) {
    touchStartX = event.touches[0].clientX;
    touchStartY = event.touches[0].clientY;
  }

  function handleTouchEnd(event) {
    const touchEndX = event.changedTouches[0].clientX;
    const touchEndY = event.changedTouches[0].clientY;
    const diffX = touchStartX - touchEndX;
    const diffY = touchStartY - touchEndY;
    const absDiffX = Math.abs(diffX);
    const absDiffY = Math.abs(diffY);

    // Check if it's a tap (minimal movement)
    if (absDiffX < 10 && absDiffY < 10) {
      // Tap in top area (header region) toggles header visibility
      if (touchStartY < 80) {
        headerVisible = !headerVisible;
      }
      return;
    }

    // Horizontal swipes for page navigation (ignore if too much vertical movement)
    if (absDiffY < 50) {
      if (diffX > 50) {
        // Swiped left - go to next page
        goNext();
      } else if (diffX < -50) {
        // Swiped right - go to previous page
        goPrev();
      }
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
    if (isTouchDevice) return;
    headerVisible = true;
  }

  function handleHeaderMouseLeave() {
    if (isTouchDevice) return;
    headerVisible = false;
  }

  function handleMouseMove(event) {
    if (isTouchDevice) return;
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

<div
  class="reader-wrapper"
  onmousemove={handleMouseMove}
  ontouchstart={handleTouchStart}
  ontouchend={handleTouchEnd}
  role="main"
>
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
    <div
      class="reader-container"
      class:pdf-mode={bookMetadata?.fileType === 'pdf'}
      bind:this={readerContainer}
      role="region"
      aria-label="Book reader"
    ></div>
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
    --background-color-hover: #e8e2d0;
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
    overscroll-behavior: none;
    touch-action: pan-x pan-y;
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
    background: var(--background-color-hover);
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
    background: var(--background-color-hover);
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
    overscroll-behavior: contain;
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

  /* PDF text layer styles for text selection */
  :global(.pdf-text-layer) {
    opacity: 0.2;
    line-height: 1.0;
    pointer-events: auto;
  }

  :global(.pdf-text-layer > span) {
    color: transparent;
    position: absolute;
    white-space: pre;
    pointer-events: all;
    transform-origin: 0 0;
  }

  :global(.pdf-text-layer ::selection) {
    background: rgba(66, 153, 225, 0.5);
  }

  :global(.pdf-text-layer ::-moz-selection) {
    background: rgba(66, 153, 225, 0.5);
  }

  /* PDF container should not scroll - content fits viewport */
  .reader-container.pdf-mode {
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    padding-top: 80px;
  }
</style>
