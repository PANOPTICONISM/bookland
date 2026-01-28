<script>
  import { onMount } from 'svelte';
  import Library from './components/Library.svelte';
  import Reader from './components/Reader.svelte';

  let currentView = $state('library');
  let selectedBookId = $state(null);

  onMount(() => {
    updateFromURL();

    window.addEventListener('popstate', updateFromURL);

    return () => {
      window.removeEventListener('popstate', updateFromURL);
    };
  });

  function updateFromURL() {
    const path = window.location.pathname;
    const match = path.match(/^\/book\/([^\/]+)$/);

    if (match) {
      selectedBookId = match[1];
      currentView = 'reader';
    } else {
      currentView = 'library';
      selectedBookId = null;
    }
  }

  function openBook(bookId) {
    selectedBookId = bookId;
    currentView = 'reader';
    window.history.pushState({}, '', `/book/${bookId}`);
  }

  function closeReader() {
    currentView = 'library';
    selectedBookId = null;
    window.history.pushState({}, '', '/');
  }
</script>

{#if currentView === 'library'}
  <Library onOpenBook={openBook} />
{:else if currentView === 'reader'}
  <Reader bookId={selectedBookId} onClose={closeReader} />
{/if}
