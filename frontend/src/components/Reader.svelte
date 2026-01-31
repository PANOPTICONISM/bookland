<script>
  import { onMount } from "svelte";
  import "foliate-js/view.js";
  import { Overlayer } from "foliate-js/overlayer.js";
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

  let fontSize = $state(20);
  const MIN_FONT_SIZE = 12;
  const MAX_FONT_SIZE = 32;

  // Store reference to EPUB content document for font size updates
  let epubContentDoc = null;

  // Store annotation colors by CFI for the draw-annotation handler
  let annotationColors = new Map();

  // Annotations
  let annotations = $state([]);
  let showAnnotationPanel = $state(false);
  let showAnnotationsList = $state(false);
  let selectedText = $state(null);
  let selectedCFI = $state(null);
  let annotationNote = $state("");
  let annotationColor = $state("yellow");
  const highlightColors = ["yellow", "green", "blue", "pink", "orange"];

  const fetchAnnotations = async () => {
    try {
      const res = await fetch(`/api/books/${bookId}/annotations`);
      if (res.ok) {
        annotations = await res.json();
      }
    } catch (err) {
      console.error("Failed to fetch annotations:", err);
    }
  };

  const createAnnotation = async () => {
    if (!selectedText || !selectedCFI) return;

    try {
      const res = await fetch(`/api/books/${bookId}/annotations`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          cfi: selectedCFI,
          text: selectedText,
          note: annotationNote,
          color: annotationColor,
        }),
      });
      if (res.ok) {
        const annotation = await res.json();
        annotations = [annotation, ...annotations];
        applyHighlight(annotation);
        closeAnnotationPanel();
      }
    } catch (err) {
      console.error("Failed to create annotation:", err);
    }
  };

  const deleteAnnotation = async (id) => {
    try {
      const res = await fetch(`/api/books/${bookId}/annotations/${id}`, {
        method: "DELETE",
      });
      if (res.ok) {
        const annotation = annotations.find((a) => a.id === id);
        annotations = annotations.filter((a) => a.id !== id);
        if (annotation) removeHighlight(annotation);
      }
    } catch (err) {
      console.error("Failed to delete annotation:", err);
    }
  };

  const applyHighlight = (annotation) => {
    if (!view || !annotation?.cfi) return;
    // Store color for this CFI so draw-annotation can use it
    annotationColors.set(annotation.cfi, annotation.color);
    // Add annotation to foliate-js view
    view.addAnnotation({ value: annotation.cfi });
  };

  const removeHighlight = (annotation) => {
    if (!view || !annotation?.cfi) return;
    annotationColors.delete(annotation.cfi);
    view.addAnnotation({ value: annotation.cfi }, true); // true = remove
  };

  const applyAllAnnotations = () => {
    if (!view || annotations.length === 0) return;
    for (const annotation of annotations) {
      applyHighlight(annotation);
    }
  };

  const closeAnnotationPanel = () => {
    showAnnotationPanel = false;
    selectedText = null;
    selectedCFI = null;
    annotationNote = "";
    annotationColor = "yellow";
  };

  // Detect touch-only device (no mouse)
  const isTouchDevice =
    typeof window !== "undefined" &&
    window.matchMedia("(pointer: coarse)").matches;

  // Set up PDF.js worker - use worker from public directory
  pdfjsLib.GlobalWorkerOptions.workerSrc = "/pdf.worker.min.mjs";

  // Save reading progress to backend (debounced)
  let saveTimeout = null;
  const saveProgress = (progress) => {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }
    saveTimeout = setTimeout(async () => {
      try {
        await fetch(`/api/books/${bookId}/progress`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ progress }),
        });
      } catch (err) {
        // Silently fail - don't interrupt reading
      }
    }, 1000); // Wait 1 second after last change before saving
  };

  onMount(async () => {
    const savedFontSize = localStorage.getItem("readerFontSize");
    if (savedFontSize) {
      fontSize = parseInt(savedFontSize, 10);
    }

    try {
      // Fetch book metadata first to determine file type
      const metadataResponse = await fetch(`/api/books/${bookId}`);
      if (!metadataResponse.ok) {
        throw new Error("Failed to load book metadata");
      }
      bookMetadata = await metadataResponse.json();

      // Fetch annotations first (before book loads)
      await fetchAnnotations();

      // Then fetch the book file
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
      epubContentDoc = null;
    };
  });

  // Formats supported by foliate-js
  const foliateFormats = ["epub", "mobi", "fb2", "cbz"];

  $effect(() => {
    if (
      readerContainer &&
      bookBlob &&
      bookMetadata &&
      foliateFormats.includes(bookMetadata.fileType) &&
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
          saveProgress(JSON.stringify({ type: bookMetadata.fileType, cfi, fraction }));
        }
        applyAllAnnotations();
      });

      view.addEventListener("load", (e) => {
        try {
          const doc = e.detail?.doc;
          if (doc && doc.documentElement && doc.head) {
            // Store reference for later font size updates
            epubContentDoc = doc;

            const isDark = document.documentElement.classList.contains("dark");
            const style = doc.createElement("style");
            style.id = "bookland-font-style";
            style.textContent = `
              p, div, span, li, td, th {
                font-size: ${fontSize}px !important;
              }
              ${
                isDark
                  ? `
              * {
                color: #e2e8f0 !important;
              }
              a {
                color: #63b3ed !important;
              }
              `
                  : ""
              }
            `;
            doc.head.appendChild(style);

            let selectionTimeout = null;
            doc.addEventListener("selectionchange", () => {
              if (selectionTimeout) clearTimeout(selectionTimeout);
              selectionTimeout = setTimeout(() => {
                const selection = doc.getSelection();
                if (selection && selection.rangeCount > 0 && !selection.isCollapsed) {
                  const range = selection.getRangeAt(0);
                  const text = selection.toString().trim();
                  if (text && text.length > 0) {
                    selectedText = text;
                    try {
                      const cfi = view.getCFI(e.detail.index, range);
                      selectedCFI = cfi || `section-${e.detail.index}`;
                    } catch {
                      selectedCFI = `section-${e.detail.index}`;
                    }
                    showAnnotationPanel = true;
                  }
                }
              }, 200);
            });
          }
        } catch (err) {
          // Ignore styling errors
        }
      });

      // Customize highlight drawing with annotation colors
      view.addEventListener("draw-annotation", (e) => {
        const { draw, annotation } = e.detail;
        const cfi = annotation.value;
        const color = annotationColors.get(cfi) || "yellow";
        // Use Overlayer.highlight with the annotation's color
        draw(Overlayer.highlight, { color });
      });

      // Create a File object from the blob with proper filename
      // Foliate-js needs the filename to determine file type
      const extMap = { epub: ".epub", mobi: ".mobi", fb2: ".fb2", cbz: ".cbz" };
      const mimeMap = {
        epub: "application/epub+zip",
        mobi: "application/x-mobipocket-ebook",
        fb2: "application/x-fictionbook+xml",
        cbz: "application/vnd.comicbook+zip",
      };
      const ext = extMap[bookMetadata.fileType] || ".epub";
      const mime = mimeMap[bookMetadata.fileType] || "application/epub+zip";
      const filename = bookMetadata.title
        ? `${bookMetadata.title}${ext}`
        : `book${ext}`;
      const file = new File([bookBlob], filename, { type: mime });

      // Open the book and restore saved progress or start from beginning
      view
        .open(file)
        .then(() => {
          if (bookMetadata.readingProgress) {
            try {
              const progress = JSON.parse(bookMetadata.readingProgress);
              if (foliateFormats.includes(progress.type) && progress.cfi) {
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
          error = "Failed to open book: " + err.message;
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

  const loadPDF = async () => {
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
          if (progress.type === "pdf" && progress.page) {
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
  };

  const renderPDFPage = async (pageNum) => {
    if (!pdfDoc || !readerContainer) return;

    currentPage = pageNum;
    currentLocation = pageNum;

    const page = await pdfDoc.getPage(pageNum);

    const containerHeight = window.innerHeight;
    const containerWidth = Math.min(window.innerWidth, 900);
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

    saveProgress(JSON.stringify({ type: "pdf", page: pageNum, totalPages }));
  };

  const goNext = () => {
    if (foliateFormats.includes(bookMetadata?.fileType)) {
      view?.next();
    } else if (bookMetadata?.fileType === "pdf") {
      if (currentPage < totalPages) {
        renderPDFPage(currentPage + 1);
      }
    }
  };

  const goPrev = () => {
    if (foliateFormats.includes(bookMetadata?.fileType)) {
      view?.prev();
    } else if (bookMetadata?.fileType === "pdf") {
      if (currentPage > 1) {
        renderPDFPage(currentPage - 1);
      }
    }
  };

  let touchStartX = 0;
  let touchStartY = 0;

  const handleKeyPress = (event) => {
    if (event.key === "ArrowRight") {
      goNext();
    } else if (event.key === "ArrowLeft") {
      goPrev();
    } else if (event.key === "Escape") {
      onClose();
    }
  };

  // All touch handling on wrapper for full-screen coverage
  const handleTouchStart = (event) => {
    touchStartX = event.touches[0].clientX;
    touchStartY = event.touches[0].clientY;
  };

  const handleTouchEnd = (event) => {
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
  };

  const startHideTimer = () => {
    if (hideTimeout) {
      clearTimeout(hideTimeout);
    }
    hideTimeout = setTimeout(() => {
      headerVisible = false;
    }, 3000);
  };

  const handleHeaderMouseEnter = () => {
    if (isTouchDevice) return;
    headerVisible = true;
  };

  const handleHeaderMouseLeave = () => {
    if (isTouchDevice) return;
    headerVisible = false;
  };

  const handleMouseMove = (event) => {
    if (isTouchDevice) return;
    if (event.clientY < 60 && !headerVisible) {
      headerVisible = true;
    }
  };

  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen();
      isFullscreen = true;
    } else {
      document.exitFullscreen();
      isFullscreen = false;
    }
  };

  const increaseFontSize = () => {
    if (fontSize < MAX_FONT_SIZE) {
      fontSize += 2;
      localStorage.setItem("readerFontSize", fontSize.toString());
    }
  };

  const decreaseFontSize = () => {
    if (fontSize > MIN_FONT_SIZE) {
      fontSize -= 2;
      localStorage.setItem("readerFontSize", fontSize.toString());
    }
  };

  $effect(() => {
    const currentSize = fontSize;
    // Text-based formats support font size changes (not cbz which is images)
    const textFormats = ["epub", "mobi", "fb2"];
    if (epubContentDoc && textFormats.includes(bookMetadata?.fileType)) {
      const style = epubContentDoc.getElementById("bookland-font-style");
      if (style) {
        style.textContent = style.textContent.replace(
          /font-size:\s*\d+px/g,
          `font-size: ${currentSize}px`,
        );
      }
    }
  });

  // Listen for fullscreen changes (e.g., user presses Escape)
  const handleFullscreenChange = () => {
    isFullscreen = !!document.fullscreenElement;
  };
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
    <div class="header-controls">
      {#if foliateFormats.includes(bookMetadata?.fileType)}
        <button
          class="annotations-btn"
          onclick={() => showAnnotationsList = !showAnnotationsList}
          aria-label="View annotations"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 20h9M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" />
          </svg>
          {#if annotations.length > 0}
            <span class="annotation-count">{annotations.length}</span>
          {/if}
        </button>
      {/if}
      {#if ["epub", "mobi", "fb2"].includes(bookMetadata?.fileType)}
        <div class="font-size-controls">
          <button
            class="font-btn"
            onclick={decreaseFontSize}
            disabled={fontSize <= MIN_FONT_SIZE}
            aria-label="Decrease font size"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          </button>
          <span class="font-size-label">{fontSize}px</span>
          <button
            class="font-btn"
            onclick={increaseFontSize}
            disabled={fontSize >= MAX_FONT_SIZE}
            aria-label="Increase font size"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          </button>
        </div>
      {/if}
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
      class:pdf-mode={bookMetadata?.fileType === "pdf"}
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

  {#if showAnnotationPanel}
    <div class="annotation-panel">
      <div class="annotation-panel-header">
        <h3>Add Highlight</h3>
        <button class="close-panel-btn" onclick={closeAnnotationPanel} aria-label="Close annotation panel">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="selected-text">
        "{selectedText?.slice(0, 100)}{selectedText?.length > 100 ? '...' : ''}"
      </div>
      <div class="color-picker">
        {#each highlightColors as color}
          <button
            class="color-btn"
            class:selected={annotationColor === color}
            style="background-color: {color};"
            onclick={() => annotationColor = color}
            aria-label="Select {color} highlight"
          ></button>
        {/each}
      </div>
      <textarea
        class="annotation-note"
        bind:value={annotationNote}
        placeholder="Add a note (optional)..."
        rows="3"
      ></textarea>
      <button class="save-annotation-btn" onclick={createAnnotation}>
        Save Highlight
      </button>
    </div>
  {/if}

  {#if showAnnotationsList}
    <div class="annotations-list-panel">
      <div class="annotation-panel-header">
        <h3>Highlights ({annotations.length})</h3>
        <button class="close-panel-btn" onclick={() => showAnnotationsList = false} aria-label="Close annotations list">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="annotations-list">
        {#if annotations.length === 0}
          <p class="no-annotations">No highlights yet. Select text in the book to create a highlight.</p>
        {:else}
          {#each annotations as annotation (annotation.id)}
            <div class="annotation-item" style="border-left-color: {annotation.color};">
              <div class="annotation-text">"{annotation.text.slice(0, 150)}{annotation.text.length > 150 ? '...' : ''}"</div>
              {#if annotation.note}
                <div class="annotation-item-note">{annotation.note}</div>
              {/if}
              <div class="annotation-actions">
                <button
                  class="go-to-btn"
                  onclick={() => { view?.goTo(annotation.cfi); showAnnotationsList = false; }}
                >
                  Go to
                </button>
                <button
                  class="delete-btn"
                  onclick={() => deleteAnnotation(annotation.id)}
                >
                  Delete
                </button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
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
    background: rgba(245, 241, 232, 0.95);
    backdrop-filter: blur(8px);
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

  .header-controls {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .font-size-controls {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    background: rgba(0, 0, 0, 0.05);
    padding: 0.25rem;
    border-radius: 6px;
  }

  .font-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.35rem;
    border-radius: 4px;
    color: #4a5568;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s;
  }

  .font-btn:hover:not(:disabled) {
    background: rgba(0, 0, 0, 0.1);
  }

  .font-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .font-size-label {
    font-size: 0.8rem;
    color: #4a5568;
    min-width: 40px;
    text-align: center;
  }

  :global(.dark) .font-size-controls {
    background: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .font-btn {
    color: #e2e8f0;
  }

  :global(.dark) .font-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.15);
  }

  :global(.dark) .font-size-label {
    color: #e2e8f0;
  }

  .reader-container {
    flex: 1;
    overflow: auto;
    position: relative;
    max-width: 900px;
    margin: 0 auto;
    width: 100%;
    padding: 2rem;
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
    }
  }

  /* PDF text layer styles for text selection */
  :global(.pdf-text-layer) {
    opacity: 0.2;
    line-height: 1;
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
    padding: 0;
  }

  /* Dark mode styles */
  :global(.dark) .reader-wrapper {
    background: #1a202c;
  }

  :global(.dark) .reader-header {
    background: rgba(26, 32, 44, 0.95);
    border-bottom-color: #4a5568;
  }

  :global(.dark) .close-btn {
    color: #e2e8f0;
  }

  :global(.dark) .close-btn:hover {
    background: #4a5568;
  }

  :global(.dark) .fullscreen-btn {
    color: #e2e8f0;
  }

  :global(.dark) .fullscreen-btn:hover {
    background: #4a5568;
  }

  :global(.dark) .loading,
  :global(.dark) .error {
    color: #a0aec0;
  }

  :global(.dark) .progress-bar {
    color: #a0aec0;
  }

  .annotation-panel {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    background: white;
    border-top: 1px solid #e2e8f0;
    padding: 1.5rem;
    z-index: 1002;
    box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.1);
    animation: slideUp 0.2s ease-out;
  }

  @keyframes slideUp {
    from {
      transform: translateY(100%);
    }
    to {
      transform: translateY(0);
    }
  }

  .annotation-panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .annotation-panel-header h3 {
    margin: 0;
    font-size: 1.1rem;
    color: #2d3748;
  }

  .close-panel-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.25rem;
    color: #718096;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .close-panel-btn:hover {
    background: #f7fafc;
    color: #4a5568;
  }

  .selected-text {
    font-style: italic;
    color: #4a5568;
    padding: 0.75rem;
    background: #f7fafc;
    border-radius: 6px;
    margin-bottom: 1rem;
    font-size: 0.9rem;
    line-height: 1.5;
    max-height: 80px;
    overflow: hidden;
  }

  .color-picker {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .color-btn {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: transform 0.15s, border-color 0.15s;
  }

  .color-btn:hover {
    transform: scale(1.1);
  }

  .color-btn.selected {
    border-color: #2d3748;
    transform: scale(1.1);
  }

  .annotation-note {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 0.95rem;
    resize: none;
    margin-bottom: 1rem;
    font-family: inherit;
  }

  .annotation-note:focus {
    outline: none;
    border-color: #4299e1;
    box-shadow: 0 0 0 3px rgba(66, 153, 225, 0.15);
  }

  .save-annotation-btn {
    width: 100%;
    padding: 0.75rem;
    background: #4299e1;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
    font-weight: 500;
    transition: background 0.2s;
  }

  .save-annotation-btn:hover {
    background: #3182ce;
  }

  :global(.dark) .annotation-panel {
    background: #2d3748;
    border-top-color: #4a5568;
  }

  :global(.dark) .annotation-panel-header h3 {
    color: #e2e8f0;
  }

  :global(.dark) .close-panel-btn {
    color: #a0aec0;
  }

  :global(.dark) .close-panel-btn:hover {
    background: #4a5568;
    color: #e2e8f0;
  }

  :global(.dark) .selected-text {
    background: #1a202c;
    color: #e2e8f0;
  }

  :global(.dark) .color-btn.selected {
    border-color: #e2e8f0;
  }

  :global(.dark) .annotation-note {
    background: #1a202c;
    border-color: #4a5568;
    color: #e2e8f0;
  }

  :global(.dark) .annotation-note:focus {
    border-color: #4299e1;
  }

  /* Annotations Button */
  .annotations-btn {
    position: relative;
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

  .annotations-btn:hover {
    background: var(--background-color-hover);
  }

  .annotation-count {
    position: absolute;
    top: 0;
    right: 0;
    background: #4299e1;
    color: white;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.1rem 0.35rem;
    border-radius: 10px;
    min-width: 16px;
    text-align: center;
  }

  :global(.dark) .annotations-btn {
    color: #e2e8f0;
  }

  :global(.dark) .annotations-btn:hover {
    background: #4a5568;
  }

  .annotations-list-panel {
    position: fixed;
    top: 0;
    right: 0;
    width: 350px;
    max-width: 90vw;
    height: 100vh;
    background: white;
    border-left: 1px solid #e2e8f0;
    z-index: 1002;
    box-shadow: -4px 0 20px rgba(0, 0, 0, 0.1);
    animation: slideIn 0.2s ease-out;
    display: flex;
    flex-direction: column;
  }

  @keyframes slideIn {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }

  .annotations-list-panel .annotation-panel-header {
    padding: 1rem 1.5rem;
    border-bottom: 1px solid #e2e8f0;
  }

  .annotations-list {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
  }

  .no-annotations {
    color: #718096;
    text-align: center;
    padding: 2rem 1rem;
    font-size: 0.9rem;
  }

  .annotation-item {
    padding: 0.75rem;
    border-left: 4px solid yellow;
    background: #f7fafc;
    border-radius: 0 6px 6px 0;
    margin-bottom: 0.75rem;
  }

  .annotation-text {
    font-size: 0.9rem;
    color: #4a5568;
    line-height: 1.5;
    font-style: italic;
  }

  .annotation-item-note {
    font-size: 0.85rem;
    color: #718096;
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px dashed #e2e8f0;
  }

  .annotation-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .go-to-btn,
  .delete-btn {
    padding: 0.35rem 0.75rem;
    font-size: 0.8rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .go-to-btn {
    background: #4299e1;
    color: white;
  }

  .go-to-btn:hover {
    background: #3182ce;
  }

  .delete-btn {
    background: #fc8181;
    color: white;
  }

  .delete-btn:hover {
    background: #f56565;
  }

  :global(.dark) .annotations-list-panel {
    background: #2d3748;
    border-left-color: #4a5568;
  }

  :global(.dark) .annotations-list-panel .annotation-panel-header {
    border-bottom-color: #4a5568;
  }

  :global(.dark) .no-annotations {
    color: #a0aec0;
  }

  :global(.dark) .annotation-item {
    background: #1a202c;
  }

  :global(.dark) .annotation-text {
    color: #e2e8f0;
  }

  :global(.dark) .annotation-item-note {
    color: #a0aec0;
    border-top-color: #4a5568;
  }
</style>
