<script>
  let {
    selectedText,
    annotationNote = $bindable(""),
    annotationColor = $bindable("yellow"),
    onSave,
    onClose,
  } = $props();

  const highlightColors = ["yellow", "green", "blue", "pink", "orange"];
</script>

<div class="annotation-panel">
  <div class="annotation-panel-header">
    <h3>Add Highlight</h3>
    <button class="close-panel-btn" onclick={onClose} aria-label="Close annotation panel">
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
  <button class="save-annotation-btn" onclick={onSave}>
    Save Highlight
  </button>
</div>

<style>
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
</style>
