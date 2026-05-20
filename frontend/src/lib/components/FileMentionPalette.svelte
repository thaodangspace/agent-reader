<script>
  import { searchFS, debouncedSearch } from '$lib/api/fs.js';
  import { Folder, CornerUpLeft, FileCode, Settings, FileText, FileImage, Terminal, File } from '@lucide/svelte';

  let { sessionId, input, onFileSelect, onMentionClose } = $props();

  let entries = $state([]);
  let loading = $state(false);
  let selectedIndex = $state(0);
  let searchQuery = $state('');

  function escapeHTML(str) {
    if (typeof str !== 'string') str = str == null ? '' : String(str);
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/</g, '&gt;');
  }

  function getMentionQuery() {
    const atIdx = input.lastIndexOf('@');
    if (atIdx === -1) return '';
    const afterAt = input.slice(atIdx + 1);
    const spaceIdx = afterAt.indexOf(' ');
    if (spaceIdx !== -1) return afterAt.slice(0, spaceIdx);
    return afterAt;
  }

  function entryIcon(entry) {
    if (entry.is_dir) return Folder;
    const ext = entry.name.split('.').pop().toLowerCase();
    const codeExts = ['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'rs', 'rb', 'java', 'c', 'cpp', 'h', 'html', 'css', 'makefile'];
    const configExts = ['json', 'yaml', 'yml', 'toml', 'dockerfile'];
    const textExts = ['md', 'txt'];
    const imgExts = ['png', 'jpg', 'jpeg', 'gif', 'svg'];
    const scriptExts = ['sh', 'bash', 'zsh'];

    if (codeExts.includes(ext)) return FileCode;
    if (configExts.includes(ext)) return Settings;
    if (textExts.includes(ext)) return FileText;
    if (imgExts.includes(ext)) return FileImage;
    if (scriptExts.includes(ext)) return Terminal;
    return File;
  }
 
  $effect(() => {
    const _ = input;
    const query = getMentionQuery();
    if (!query) {
      entries = [];
      loading = false;
      return;
    }
    if (query === searchQuery && entries.length > 0) return;
    searchQuery = query;
    loading = true;
    debouncedSearch(query, (result) => {
      if (result.success) {
        entries = (result.entries || []).slice(0, 15);
      } else {
        entries = [];
      }
      loading = false;
      selectedIndex = 0;
    });
  });
 
  let show = $derived(entries.length > 0 || loading);
 
  function selectEntry(entry) {
    onFileSelect(entry);
  }
 
  function navigateUp() {
    if (entries.length === 0) return;
    selectedIndex = (selectedIndex - 1 + entries.length) % entries.length;
  }
 
  function navigateDown() {
    if (entries.length === 0) return;
    selectedIndex = (selectedIndex + 1) % entries.length;
  }
 
  function handleKeydown(e) {
    if (!show) return false;
    if (e.key === 'ArrowDown') {
      e.preventDefault(); e.stopPropagation(); navigateDown(); return true;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault(); e.stopPropagation(); navigateUp(); return true;
    }
    if (e.key === 'Enter' && !e.shiftKey && entries.length > 0) {
      e.preventDefault(); e.stopPropagation(); selectEntry(entries[selectedIndex]); return true;
    }
    if (e.key === 'Escape') {
      e.preventDefault(); e.stopPropagation(); onMentionClose(); return true;
    }
    if (e.key === 'Tab' && entries.length > 0) {
      e.preventDefault(); e.stopPropagation(); selectEntry(entries[selectedIndex]); return true;
    }
    return false;
  }
 
  export { handleKeydown, show };
</script>
 
{#if show}
  <div
    class="file-mention-palette absolute bottom-full left-0 right-0 mb-1 bg-ctp-mantle border border-ctp-surface0 rounded-lg shadow-lg overflow-hidden z-50 max-h-60 overflow-y-auto"
  >
    {#if loading}
      <div class="px-3 py-3 text-center text-[11px] text-ctp-overlay0">
        <div class="w-3 h-3 border-2 border-ctp-blue border-t-transparent rounded-full animate-spin mx-auto mb-1"></div>
        Searching files...
      </div>
    {:else if entries.length === 0}
      <div class="px-3 py-3 text-center text-[11px] text-ctp-overlay0">
        No files found
      </div>
    {:else}
      <div class="px-3 py-1.5 border-b border-ctp-surface0/50 text-[10px] text-ctp-overlay0 flex items-center justify-between">
        <span>Files —  navigate,  select, tab autocomplete</span>
        <span>{entries.length} found</span>
      </div>
      {#each entries as entry, i}
        {@const Icon = entryIcon(entry)}
        <button
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 transition-colors hover:bg-ctp-surface0/70 cursor-pointer"
          class:bg-ctp-surface0={i === selectedIndex}
          onclick={() => selectEntry(entry)}
          onmouseenter={() => selectedIndex = i}
        >
          <span class="text-xs shrink-0 flex items-center justify-center text-ctp-overlay1">
            <Icon size={14} />
          </span>
          <div class="flex-1 min-w-0">
            <div class="text-xs font-mono text-ctp-text truncate">{escapeHTML(entry.name)}</div>
            <div class="text-[9px] text-ctp-overlay1 truncate">{escapeHTML(entry.path)}</div>
          </div>
          {#if entry.is_dir}
            <span class="text-[9px] text-ctp-overlay1 shrink-0">dir</span>
          {:else if entry.size}
            <span class="text-[9px] text-ctp-overlay1 shrink-0">{Math.round(entry.size / 1024)}KB</span>
          {/if}
        </button>
      {/each}
    {/if}
  </div>
{/if}
