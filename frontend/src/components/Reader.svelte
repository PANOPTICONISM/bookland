<script>
  import { onMount } from 'svelte';
  import 'foliate-js/view.js';
  import * as pdfjsLib from 'pdfjs-dist';

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

  // Set up PDF.js worker - use worker from public directory
  pdfjsLib.GlobalWorkerOptions.workerSrc = '/pdf.worker.min.mjs';

  onMount(async () => {
    try {
      // Fetch book metadata first to determine file type
      const metadataResponse = await fetch(`/api/books/${bookId}`);
      if (!metadataResponse.ok) throw new Error('Failed to load book metadata');
      bookMetadata = await metadataResponse.json();

      // Fetch the book file
      const fileResponse = await fetch(`/api/books/${bookId}/file`);
      if (!fileResponse.ok) throw new Error('Failed to load book');
      bookBlob = await fileResponse.blob();

      loading = false;
      startHideTimer();
    } catch (err) {
      error = err.message;
      loading = false;
    }

    return () => {
      if (hideTimeout) clearTimeout(hideTimeout);
      if (view) view.remove();
      if (pdfDoc) pdfDoc.destroy();
    };
  });

  // Effect for EPUB rendering
  $effect(() => {
    if (readerContainer && bookBlob && bookMetadata && bookMetadata.fileType === 'epub' && !view) {
      // Create foliate-view web component
      view = document.createElement('foliate-view');
      view.style.width = '100%';
      view.style.height = '100%';
      readerContainer.appendChild(view);

      // Listen for location changes
      view.addEventListener('relocate', (e) => {
        const fraction = e.detail.fraction;
        if (fraction !== undefined) {
          currentLocation = Math.round(fraction * 100);
          totalLocations = 100;
        }
      });

      // Create a File object from the blob with proper filename
      // Foliate-js needs the filename to determine file type
      const filename = bookMetadata.title ? `${bookMetadata.title}.epub` : 'book.epub';
      const file = new File([bookBlob], filename, { type: 'application/epub+zip' });

      // Open the book and navigate to the start
      view.open(file).then(() => {
        // Navigate to the beginning of the book
        view.goTo(0);
      }).catch(err => {
        error = 'Failed to open EPUB: ' + err.message;
      });
    }
  });

  // Effect for PDF rendering
  $effect(() => {
    if (readerContainer && bookBlob && bookMetadata && bookMetadata.fileType === 'pdf' && !pdfDoc) {
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
      error = 'Failed to load PDF: ' + err.message;
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
    readerContainer.innerHTML = '';

    // Create canvas
    const canvas = document.createElement('canvas');
    const context = canvas.getContext('2d');
    canvas.height = viewport.height;
    canvas.width = viewport.width;
    canvas.style.display = 'block';
    canvas.style.margin = '0 auto';

    readerContainer.appendChild(canvas);

    // Render PDF page
    await page.render({
      canvasContext: context,
      viewport: viewport
    }).promise;
  }

  function goNext() {
    if (bookMetadata?.fileType === 'epub') {
      view?.next();
    } else if (bookMetadata?.fileType === 'pdf') {
      if (currentPage < totalPages) {
        renderPDFPage(currentPage + 1);
      }
    }
  }

  function goPrev() {
    if (bookMetadata?.fileType === 'epub') {
      view?.prev();
    } else if (bookMetadata?.fileType === 'pdf') {
      if (currentPage > 1) {
        renderPDFPage(currentPage - 1);
      }
    }
  }

  function handleKeyPress(event) {
    if (event.key === 'ArrowRight') goNext();
    else if (event.key === 'ArrowLeft') goPrev();
    else if (event.key === 'Escape') onClose();
  }

  function startHideTimer() {
    if (hideTimeout) clearTimeout(hideTimeout);
    hideTimeout = setTimeout(() => {
      headerVisible = false;
    }, 3000);
  }

  function handleHeaderMouseEnter() {
    if (hideTimeout) clearTimeout(hideTimeout);
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
</script>

<svelte:window on:keydown={handleKeyPress} />

<div class="reader-wrapper" onmousemove={handleMouseMove} role="main">
  <div
    class="reader-header"
    class:hidden={!headerVisible}
    onmouseenter={handleHeaderMouseEnter}
    onmouseleave={handleHeaderMouseLeave}
    role="banner"
  >
    <button class="close-btn" onclick={onClose}>
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M19 12H5M12 19l-7-7 7-7"/>
      </svg>
      Back to Library
    </button>
    {#if totalLocations > 0}
      <div class="progress">
        {#if bookMetadata?.fileType === 'pdf'}
          <span>Page {currentLocation} of {totalLocations}</span>
        {:else}
          <span>{Math.round((currentLocation / totalLocations) * 100)}%</span>
        {/if}
      </div>
    {/if}
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

    <div class="navigation">
      <button class="nav-btn" onclick={goPrev} aria-label="Previous page">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>
      <button class="nav-btn" onclick={goNext} aria-label="Next page">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
      </button>
    </div>
  {/if}
</div>

<style>
  .reader-wrapper {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: #fefefe;
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
    background: white;
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

  .progress {
    font-size: 0.9rem;
    color: #718096;
    font-weight: 500;
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
    to { transform: rotate(360deg); }
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

  .navigation {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    display: flex;
    gap: 1rem;
  }

  .nav-btn {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: white;
    border: 1px solid #e2e8f0;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #4a5568;
    transition: all 0.2s;
  }

  .nav-btn:hover {
    background: #4299e1;
    color: white;
    border-color: #4299e1;
    box-shadow: 0 4px 12px rgba(66, 153, 225, 0.3);
  }

  @media (max-width: 768px) {
    .reader-container {
      padding: 1rem;
      padding-top: 4rem;
    }

    .navigation {
      bottom: 1rem;
      right: 1rem;
    }
  }
</style>
