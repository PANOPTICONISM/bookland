<script>
  import { onMount } from 'svelte';
  import 'foliate-js/view.js';

  export let bookId;
  export let onClose;

  let readerContainer;
  let view = null;
  let loading = true;
  let error = null;
  let currentLocation = 0;
  let totalLocations = 0;

  onMount(async () => {
    try {
      const response = await fetch(`/api/books/${bookId}/file`);
      if (!response.ok) throw new Error('Failed to load book');

      const blob = await response.blob();

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

      // Open the book
      await view.open(blob);
      loading = false;
    } catch (err) {
      error = err.message;
      loading = false;
    }

    return () => {
      if (view) view.remove();
    };
  });

  function goNext() {
    view?.next();
  }

  function goPrev() {
    view?.prev();
  }

  function handleKeyPress(event) {
    if (event.key === 'ArrowRight') goNext();
    else if (event.key === 'ArrowLeft') goPrev();
    else if (event.key === 'Escape') onClose();
  }
</script>

<svelte:window on:keydown={handleKeyPress} />

<div class="reader-wrapper">
  <div class="reader-header">
    <button class="close-btn" on:click={onClose}>
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M19 12H5M12 19l-7-7 7-7"/>
      </svg>
      Back to Library
    </button>
    {#if totalLocations > 0}
      <div class="progress">
        <span>{Math.round((currentLocation / totalLocations) * 100)}%</span>
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
      <button on:click={onClose}>Go Back</button>
    </div>
  {:else}
    <div class="reader-container" bind:this={readerContainer}></div>

    <div class="navigation">
      <button class="nav-btn" on:click={goPrev}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>
      <button class="nav-btn" on:click={goNext}>
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
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 2rem;
    background: white;
    border-bottom: 1px solid #e2e8f0;
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
    overflow: hidden;
    position: relative;
    max-width: 900px;
    margin: 0 auto;
    width: 100%;
    padding: 2rem;
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
    }

    .navigation {
      bottom: 1rem;
      right: 1rem;
    }
  }
</style>
