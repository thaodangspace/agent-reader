<script>
  import { browseFS, searchFS } from '$lib/api/fs.js';
  import { Folder, CornerUpLeft, FileCode, Settings, FileText, FileImage, Terminal, File } from '@lucide/svelte';

  let { value, onSelect, onClose } = $props();

  let entries = $state([]);
  let loading = $state(false);
  let selectedIndex = $state(0);
  let currentDir = $state('');
  let showPicker = $state(false);
  let pickerTop = $state(0);
  let pickerLeft = $state(0);
  let pickerWidth = $state(0);

  // Calculate position relative to the input element
  function updatePosition() {
    const inputEl = document.querySelector('.path-picker-input');
    if (!inputEl) return;
    const rect = inputEl.getBoundingClientRect();
    pickerTop = rect.bottom + 4; // 4px gap below input
    pickerLeft = rect.left;
    pickerWidth = rect.width;
  }

  // Build an "Up" entry
  function buildUpEntry(dir) {
    if (!dir || dir === '.' || dir === '/') return null;
    const parent = dir.replace(/\/[^/]+\/?$/, '') || '/';
    return { name: '..', path: parent, is_dir: true };
  }

  function resolveCurrentDir(input) {
    if (!input || input.trim() === '') return '.';
    const trimmed = input.trim();

    // If it ends with /, it's clearly a directory path to browse
    if (trimmed.endsWith('/')) return trimmed;

    // Try to extract directory part
    const lastSlash = trimmed.lastIndexOf('/');
    if (lastSlash <= 0) return '.'; // no meaningful directory

    return trimmed.substring(0, lastSlash + 1);
  }

  async function loadEntries(dirPath) {
    if (!dirPath) return;
    loading = true;
    showPicker = true;
    updatePosition();
    try {
      const result = await browseFS(dirPath);
      if (result.success) {
        const upEntry = buildUpEntry(dirPath);
        entries = upEntry ? [upEntry, ...(result.entries || [])] : (result.entries || []);
        selectedIndex = 0;
      } else {
        entries = [];
      }
    } catch (e) {
      console.error('Failed to browse:', e);
      entries = [];
    } finally {
      loading = false;
    }
  }

  async function handleValueChange() {
    const trimmed = value.trim();

    // Empty input → show allowed roots
    if (!trimmed) {
      currentDir = '.';
      loadEntries('.');
      return;
    }

    const dir = resolveCurrentDir(trimmed);

    // If the directory changed, browse it
    if (dir !== currentDir) {
      currentDir = dir;
      loadEntries(dir);
      return;
    }

    // Same directory but maybe user is typing a subdirectory name
    const lastSlash = trimmed.lastIndexOf('/');
    if (lastSlash > 0) {
      const partial = trimmed.slice(lastSlash + 1).toLowerCase();
      if (partial) {
        const filtered = entries.filter(e =>
          e.name.toLowerCase().includes(partial) ||
          e.name.toLowerCase().startsWith(partial)
        );
        if (filtered.length > 0 && filtered.length < entries.length) {
          entries = filtered;
          selectedIndex = 0;
          showPicker = true;
          updatePosition();
        } else if (entries.length > 0) {
          showPicker = true;
          updatePosition();
        } else {
          showPicker = false;
        }
        return;
      }
    }

    if (entries.length > 0) {
      showPicker = true;
      updatePosition();
    } else {
      showPicker = false;
    }
  }

  // React to value changes
  $effect(() => {
    const _ = value;
    handleValueChange();
  });

  function selectEntry(entry, keepOpen = false) {
    if (entry.is_dir) {
      if (keepOpen) {
        // Tab: navigate into directory and keep browsing
        value = entry.path.endsWith('/') ? entry.path : entry.path + '/';
        return;
      }
      // Enter: fill directory path and close picker
      value = entry.path.endsWith('/') ? entry.path : entry.path + '/';
      showPicker = false;
      onSelect(entry.path);
    } else {
      // For files, close and submit
      onSelect(entry.path);
    }
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
    if (!showPicker && entries.length === 0 && !loading) return false;

    if (e.key === 'ArrowDown') {
      e.preventDefault(); e.stopPropagation(); navigateDown(); return true;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault(); e.stopPropagation(); navigateUp(); return true;
    }
    if (e.key === 'Enter' && entries.length > 0) {
      e.preventDefault(); e.stopPropagation();
      selectEntry(entries[selectedIndex], false); // Enter fills and closes
      return true;
    }
    if (e.key === 'Escape') {
      e.preventDefault(); e.stopPropagation();
      showPicker = false;
      return true;
    }
    if (e.key === 'Tab' && entries.length > 0) {
      e.preventDefault(); e.stopPropagation();
      selectEntry(entries[selectedIndex], true); // Tab fills and keeps open
      return true;
    }
    return false;
  }

  function entryIcon(entry) {
    if (entry.is_dir) return entry.name === '..' ? CornerUpLeft : Folder;
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
 
  let show = $derived(showPicker || loading);
 
  // Update position on window resize, close on scroll
  $effect(() => {
    if (!showPicker) return;
    updatePosition();
    const onResize = () => updatePosition();
    const onScroll = () => { showPicker = false; };
    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onScroll, true);
    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('scroll', onScroll, true);
    };
  });
 
  export { handleKeydown, show };
</script>
 
{#if show}
  <div
    class="path-picker fixed bg-ctp-mantle border border-ctp-surface0 rounded-lg shadow-lg overflow-hidden z-[9999] max-h-60 overflow-y-auto"
    style="top: {pickerTop}px; left: {pickerLeft}px; width: {pickerWidth}px;"
  >
    {#if loading}
      <div class="px-3 py-3 text-center text-[11px] text-ctp-overlay0">
        <div class="w-3 h-3 border-2 border-ctp-blue border-t-transparent rounded-full animate-spin mx-auto mb-1"></div>
        Loading...
      </div>
    {:else if entries.length === 0}
      <div class="px-3 py-3 text-center text-[11px] text-ctp-overlay0">
        No results
      </div>
    {:else}
      <div class="px-3 py-1.5 border-b border-ctp-surface0/50 text-[10px] text-ctp-overlay0 flex items-center justify-between">
        <span>{currentDir === '.' ? 'Allowed roots' : currentDir} — ↑↓ navigate, ↵ select, tab autocomplete</span>
        <span>{entries.length} items</span>
      </div>
      {#each entries as entry, i}
        {@const Icon = entryIcon(entry)}
        <button
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 transition-colors hover:bg-ctp-surface0/70 cursor-pointer"
          class:bg-ctp-surface0={i === selectedIndex}
          onclick={() => selectEntry(entry, false)}
          onmouseenter={() => selectedIndex = i}
        >
          <span class="text-xs shrink-0 flex items-center justify-center text-ctp-overlay1">
            <Icon size={14} />
          </span>
          <div class="flex-1 min-w-0">
            <div class="text-xs font-mono truncate {entry.is_dir ? 'text-ctp-blue font-semibold' : 'text-ctp-text'}">{entry.name}</div>
            {#if entry.size}
              <div class="text-[9px] text-ctp-overlay1">{Math.round(entry.size / 1024)}KB</div>
            {/if}
          </div>
          {#if entry.is_dir && entry.name !== '..'}
            <span class="text-[9px] text-ctp-overlay1 shrink-0">dir ↩</span>
          {/if}
        </button>
      {/each}
    {/if}
  </div>
{/if}
