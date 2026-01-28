<script>
  import Library from './components/Library.svelte';
  import Reader from './components/Reader.svelte';

  let currentView = 'library';
  let selectedBookId = null;

  function openBook(bookId) {
    selectedBookId = bookId;
    currentView = 'reader';
  }

  function closeReader() {
    currentView = 'library';
    selectedBookId = null;
  }
</script>

{#if currentView === 'library'}
  <Library onOpenBook={openBook} />
{:else if currentView === 'reader'}
  <Reader bookId={selectedBookId} onClose={closeReader} />
{/if}
