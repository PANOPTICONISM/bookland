<script>
  let { annotations, onGoTo, onDelete, onClose } = $props();
</script>

<div class="annotations-list-panel">
  <div class="panel-header">
    <h3>Highlights ({annotations.length})</h3>
    <button class="close-panel-btn" onclick={onClose} aria-label="Close annotations list">
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
            <button class="go-to-btn" onclick={() => onGoTo(annotation.cfi)}>
              Go to
            </button>
            <button class="delete-btn" onclick={() => onDelete(annotation.id)}>
              Delete
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
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

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid #e2e8f0;
  }

  .panel-header h3 {
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

  :global(.dark) .panel-header {
    border-bottom-color: #4a5568;
  }

  :global(.dark) .panel-header h3 {
    color: #e2e8f0;
  }

  :global(.dark) .close-panel-btn {
    color: #a0aec0;
  }

  :global(.dark) .close-panel-btn:hover {
    background: #4a5568;
    color: #e2e8f0;
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
