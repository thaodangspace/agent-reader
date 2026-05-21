<script>
  import { formatText, escapeHTML } from '$lib/utils/markdown.js';
  import { extractImagePaths } from '$lib/utils/images.js';
  import { imageViewUrl, translateText } from '$lib/api/rpc.js';
  import ThinkingBlock from './ThinkingBlock.svelte';
  import ToolCallBlock from './ToolCallBlock.svelte';
  import ImageViewer from './ImageViewer.svelte';
  import { ChevronRight, ChevronDown, Wrench } from '@lucide/svelte';

  let { msg } = $props();

  // Derive values from msg prop so they auto-update during streaming
  let rawText = $derived(msg.rawText || '');
  let thinking = $derived(msg.thinking || '');
  let toolCalls = $derived(msg.toolCalls || []);
  let isStreaming = $derived(msg.isStreaming || false);
  let hasTools = $derived(toolCalls.length > 0);

  // Detect local image paths mentioned in the assistant's response
  let imagePaths = $derived(extractImagePaths(rawText));
  let imageList = $derived(imagePaths.map(path => ({ src: imageViewUrl(path) })));

  // Tool section collapse state
  let toolsCollapsed = $state(false);

  // Image viewer state
  let showViewer = $state(false);
  let viewerStartIndex = $state(0);

  function openViewer(index) {
    viewerStartIndex = index;
    showViewer = true;
  }

  // Translation state
  let showTranslateBtn = $state(false);
  let isTranslating = $state(false);
  let translatedText = $state('');
  let showTranslation = $state(false);
  let translateError = $state('');

  async function handleTranslate() {
    if (isTranslating || !rawText) return;
    isTranslating = true;
    translateError = '';

    try {
      const resp = await translateText(rawText, 'vi');
      if (resp.success) {
        translatedText = resp.translated;
        showTranslation = true;
      } else {
        translateError = resp.error || 'Translation failed';
      }
    } catch (e) {
      translateError = e.message;
    } finally {
      isTranslating = false;
    }
  }

  function toggleTranslation() {
    if (showTranslation) {
      showTranslation = false;
    } else {
      if (translatedText) {
        showTranslation = true;
      } else {
        handleTranslate();
      }
    }
  }

  function handleMouseEnter() {
    if (!isStreaming && rawText) {
      showTranslateBtn = true;
    }
  }

  function handleMouseLeave() {
    showTranslateBtn = false;
  }
</script>

<div class="flex flex-col items-start animate-fadeIn">
  {#if imageList.length > 0}
    <div class="flex flex-wrap gap-2 mb-2 max-w-[75%]">
      {#each imageList as img, i}
        <img
          src={img.src}
          alt=""
          class="max-w-48 max-h-48 rounded-lg object-contain border border-ctp-crust cursor-pointer hover:opacity-90 transition-opacity"
          loading="lazy"
          onclick={() => openViewer(i)}
        />
      {/each}
    </div>
  {/if}

  <div
    class="px-4 py-2.5 rounded-2xl text-sm leading-relaxed break-words max-w-[75%] assistant-bubble relative group"
    style="background:#fff9f9; border-bottom-left-radius: 4px;"
    onmouseenter={handleMouseEnter}
    onmouseleave={handleMouseLeave}
  >
    <!-- Translate button (shown on hover for non-streaming text responses) -->
    {#if rawText && !isStreaming}
      <div
        class="absolute -top-3 right-2 flex items-center gap-1 transition-opacity duration-200 z-10"
        class:opacity-0={!showTranslateBtn && !isTranslating && !showTranslation}
        class:opacity-100={showTranslateBtn || isTranslating || showTranslation}
      >
        {#if translateError}
          <span class="text-[10px] text-ctp-red bg-ctp-red/10 px-1.5 py-0.5 rounded">{translateError}</span>
        {/if}
        {#if showTranslation}
          <button
            class="text-[10px] px-1.5 py-0.5 rounded cursor-pointer bg-ctp-blue/10 text-ctp-blue hover:bg-ctp-blue/20 transition-colors"
            title="Show original"
            onclick={toggleTranslation}
          >
            🇬🇧 Original
          </button>
        {:else}
          <button
            class="text-[10px] px-1.5 py-0.5 rounded cursor-pointer bg-ctp-blue/10 text-ctp-blue hover:bg-ctp-blue/20 transition-colors flex items-center gap-1"
            title="Translate to Vietnamese"
            onclick={toggleTranslation}
            disabled={isTranslating}
          >
            {#if isTranslating}
              <span class="w-2.5 h-2.5 border border-ctp-blue border-t-transparent rounded-full animate-spin"></span>
              <span>Translating...</span>
            {:else}
              <span>🇻🇳 Translate</span>
            {/if}
          </button>
        {/if}
      </div>
    {/if}

    <!-- Thinking block -->
    {#if thinking}
      <ThinkingBlock content={thinking} />
    {/if}

    <!-- Tools section (grouped hierarchy) -->
    {#if hasTools}
      <div
        class="rounded-lg overflow-hidden border border-ctp-crust mb-2"
        style="background:color-mix(in srgb, #135ce0 4%, #ffffff)"
      >
        <button
          class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer"
          onclick={() => toolsCollapsed = !toolsCollapsed}
        >
          <span class="flex items-center">
            {#if toolsCollapsed}
              <ChevronRight size={12} />
            {:else}
              <ChevronDown size={12} />
            {/if}
          </span>
          <Wrench size={14} class="text-ctp-blue" />
          <span class="font-semibold text-ctp-blue">Tools</span>
          <span class="text-ctp-overlay0 text-[10px] ml-auto">{toolCalls.length} tool{toolCalls.length > 1 ? 's' : ''}</span>
        </button>
        <div class="border-t border-ctp-surface0 px-2 pt-2" class:hidden={toolsCollapsed}>
          {#each toolCalls as tc (tc.id)}
            <ToolCallBlock {tc} />
          {/each}
        </div>
      </div>
    {/if}

    <!-- Text response -->
    {#if rawText}
      {#if showTranslation && translatedText}
        <!-- Translated text -->
        <div class="prose-markdown">
          {@html formatText(translatedText)}
        </div>
      {:else}
        <!-- Original text -->
        <div class="prose-markdown">
          {@html formatText(rawText)}
        </div>
      {/if}
    {/if}
  </div>

  <div class="text-[10px] text-ctp-overlay0 mt-0.5 px-1">{msg.timestamp}</div>
</div>

{#if showViewer}
  <ImageViewer images={imageList} startIndex={viewerStartIndex} onClose={() => showViewer = false} />
{/if}
