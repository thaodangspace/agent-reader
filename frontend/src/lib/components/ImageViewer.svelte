<script>
  import { onMount, onDestroy } from 'svelte';
  import { X, ChevronLeft, ChevronRight } from '@lucide/svelte';

  let { images, startIndex = 0, onClose } = $props();
  let currentIndex = $state(startIndex);
  let visible = $state(true);

  let currentSrc = $derived(images[currentIndex]?.src || '');
  let currentAlt = $derived(images[currentIndex]?.alt || '');
  let total = $derived(images.length);

  function close() {
    visible = false;
    onClose?.();
  }

  function prev(e) {
    e.stopPropagation();
    currentIndex = (currentIndex - 1 + total) % total;
  }

  function next(e) {
    e.stopPropagation();
    currentIndex = (currentIndex + 1) % total;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowLeft') prev(e);
    else if (e.key === 'ArrowRight') next(e);
  }

  onMount(() => {
    document.addEventListener('keydown', handleKeydown);
    document.body.style.overflow = 'hidden';
  });

  onDestroy(() => {
    document.removeEventListener('keydown', handleKeydown);
    document.body.style.overflow = '';
  });
</script>

{#if visible}
  <div
    class="fixed inset-0 z-[9999] flex items-center justify-center animate-fadeIn"
    style="background: rgba(0, 0, 0, 0.85);"
    onclick={close}
  >
    <!-- Close button -->
    <button
      class="absolute top-4 right-4 z-10 w-9 h-9 flex items-center justify-center rounded-full
             bg-white/10 hover:bg-white/20 text-white transition-colors cursor-pointer"
      onclick={close}
    >
      <X class="w-4 h-4" />
    </button>

    <!-- Counter -->
    {#if total > 1}
      <div class="absolute top-4 left-4 z-10 px-3 py-1 rounded-full text-xs text-white/70 bg-white/10">
        {currentIndex + 1} / {total}
      </div>
    {/if}

    <!-- Previous button -->
    {#if total > 1}
      <button
        class="absolute left-4 z-10 w-12 h-12 flex items-center justify-center rounded-full
               bg-white/10 hover:bg-white/20 text-white transition-colors cursor-pointer"
        onclick={prev}
      >
        <ChevronLeft class="w-6 h-6" />
      </button>
    {/if}

    <!-- Next button -->
    {#if total > 1}
      <button
        class="absolute right-4 z-10 w-12 h-12 flex items-center justify-center rounded-full
               bg-white/10 hover:bg-white/20 text-white transition-colors cursor-pointer"
        onclick={next}
      >
        <ChevronRight class="w-6 h-6" />
      </button>
    {/if}

    <!-- Image -->
    <img
      src={currentSrc}
      alt={currentAlt}
      class="max-w-[90vw] max-h-[85vh] object-contain rounded-lg shadow-2xl"
      style="animation: zoomIn 0.2s ease-out;"
      onclick={(e) => e.stopPropagation()}
    />
  </div>
{/if}

<style>
  @keyframes zoomIn {
    from { transform: scale(0.9); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }
</style>
