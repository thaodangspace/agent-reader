<script>
  import { X, Copy, Check, Info } from '@lucide/svelte';
  import { formatTokens } from '$lib/utils/format.js';
  import { toast } from '$lib/stores/toast.svelte.js';

  let { show = false, sessionInfo = null, onClose } = $props();

  let copiedId = $state(false);
  let copiedCwd = $state(false);
  let copiedFile = $state(false);

  function copyToClipboard(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    } else {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.left = "-999999px";
      textArea.style.top = "-999999px";
      textArea.setAttribute("readonly", "");
      document.body.appendChild(textArea);
      
      const selectedRange = document.getSelection().rangeCount > 0 
        ? document.getSelection().getRangeAt(0) 
        : null;
        
      textArea.focus();
      textArea.select();
      
      let success = false;
      try {
        success = document.execCommand('copy');
      } catch (err) {
        console.error('Fallback copy failed', err);
      }
      
      document.body.removeChild(textArea);
      
      if (selectedRange) {
        document.getSelection().removeAllRanges();
        document.getSelection().addRange(selectedRange);
      }
      
      return success ? Promise.resolve() : Promise.reject();
    }
  }

  function handleCopy(text, type, label) {
    copyToClipboard(text)
      .then(() => {
        toast.success(`Copied ${label} to clipboard!`);
        if (type === 'id') {
          copiedId = true;
          setTimeout(() => copiedId = false, 2000);
        } else if (type === 'cwd') {
          copiedCwd = true;
          setTimeout(() => copiedCwd = false, 2000);
        } else if (type === 'file') {
          copiedFile = true;
          setTimeout(() => copiedFile = false, 2000);
        }
      })
      .catch(() => {
        toast.error(`Failed to copy ${label}.`);
      });
  }
</script>

{#if show && sessionInfo}
  <div class="fixed inset-0 z-50 flex items-center justify-center">
    <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" onclick={onClose}></div>
    <div class="relative bg-ctp-mantle border border-ctp-surface0 rounded-2xl shadow-2xl w-[500px] max-w-[95vw] animate-fadeIn overflow-hidden">
      <!-- Header -->
      <div class="px-6 pt-5 pb-4 border-b border-ctp-surface0">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-lg bg-ctp-blue/20 flex items-center justify-center text-ctp-blue">
              <Info size={16} />
            </div>
            <div>
              <h3 class="text-sm font-semibold text-ctp-text">Session Details</h3>
              <p class="text-[11px] text-ctp-overlay0 mt-0.5">Metadata and usage statistics</p>
            </div>
          </div>
          <button
            class="text-ctp-overlay0 hover:text-ctp-text transition-colors p-1 rounded-md hover:bg-ctp-surface0 flex items-center justify-center cursor-pointer"
            onclick={onClose}
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </div>

      <!-- Body -->
      <div class="px-6 py-5 space-y-4 text-xs font-mono max-h-[60vh] overflow-y-auto">
        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Session ID</span>
          <span class="col-span-2 flex items-center justify-between min-w-0 text-ctp-text">
            <span class="truncate">{sessionInfo.id}</span>
            <button
              class="ml-2 p-1 rounded-md text-ctp-overlay0 hover:text-ctp-blue hover:bg-ctp-blue/10 transition-all cursor-pointer flex items-center justify-center shrink-0"
              title="Copy Session ID"
              onclick={(e) => { e.stopPropagation(); handleCopy(sessionInfo.id, 'id', 'Session ID'); }}
            >
              {#if copiedId}
                <Check size={14} class="text-ctp-green" />
              {:else}
                <Copy size={14} />
              {/if}
            </button>
          </span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Project</span>
          <span class="col-span-2 text-ctp-text truncate">{sessionInfo.project || 'Unknown'}</span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Agent Type</span>
          <span class="col-span-2">
            <span class="px-2 py-0.5 rounded text-[10px] uppercase font-bold"
                  style="background: {sessionInfo.agent === 'claude' ? 'rgba(233,95,89,0.15)' : sessionInfo.agent === 'codex' ? 'rgba(111,66,193,0.15)' : 'rgba(19,92,224,0.15)'};
                         color: {sessionInfo.agent === 'claude' ? '#e95f59' : sessionInfo.agent === 'codex' ? '#6f42c1' : '#135ce0'}">
              {sessionInfo.agent || 'pi'}
            </span>
          </span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Active Model</span>
          <span class="col-span-2 text-ctp-text break-all">{sessionInfo.model || 'Unknown'}</span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Working Dir</span>
          <span class="col-span-2 flex items-center justify-between min-w-0 text-ctp-text">
            <span class="truncate" title={sessionInfo.cwd}>{sessionInfo.cwd || 'Unknown'}</span>
            {#if sessionInfo.cwd}
              <button
                class="ml-2 p-1 rounded-md text-ctp-overlay0 hover:text-ctp-blue hover:bg-ctp-blue/10 transition-all cursor-pointer flex items-center justify-center shrink-0"
                title="Copy Path"
                onclick={(e) => { e.stopPropagation(); handleCopy(sessionInfo.cwd, 'cwd', 'Working Directory'); }}
              >
                {#if copiedCwd}
                  <Check size={14} class="text-ctp-green" />
                {:else}
                  <Copy size={14} />
                {/if}
              </button>
            {/if}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Log File</span>
          <span class="col-span-2 flex items-center justify-between min-w-0 text-ctp-text">
            <span class="truncate font-sans text-[11px]" title={sessionInfo.file}>{sessionInfo.file || 'Unknown'}</span>
            {#if sessionInfo.file}
              <button
                class="ml-2 p-1 rounded-md text-ctp-overlay0 hover:text-ctp-blue hover:bg-ctp-blue/10 transition-all cursor-pointer flex items-center justify-center shrink-0"
                title="Copy File Path"
                onclick={(e) => { e.stopPropagation(); handleCopy(sessionInfo.file, 'file', 'File Path'); }}
              >
                {#if copiedFile}
                  <Check size={14} class="text-ctp-green" />
                {:else}
                  <Copy size={14} />
                {/if}
              </button>
            {/if}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Lines in Log</span>
          <span class="col-span-2 text-ctp-text">{sessionInfo.line_count || 0} lines</span>
        </div>

        <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
          <span class="text-ctp-overlay0 font-semibold">Token Usage</span>
          <div class="col-span-2 space-y-1 text-ctp-text">
            <div class="flex justify-between">
              <span>Input:</span>
              <span>{formatTokens(sessionInfo.input_tokens || 0)}</span>
            </div>
            <div class="flex justify-between">
              <span>Output:</span>
              <span>{formatTokens(sessionInfo.output_tokens || 0)}</span>
            </div>
            <div class="flex justify-between font-semibold border-t border-ctp-surface0/20 pt-1">
              <span>Total:</span>
              <span>{formatTokens(sessionInfo.total_tokens || 0)}</span>
            </div>
            {#if sessionInfo.context_window}
              <div class="flex justify-between text-ctp-overlay0 text-[10px] pt-0.5">
                <span>Context Window:</span>
                <span>{formatTokens(sessionInfo.context_window)}</span>
              </div>
            {/if}
          </div>
        </div>

        {#if sessionInfo.total_cost}
          <div class="grid grid-cols-3 gap-2 py-1 border-b border-ctp-surface0/30">
            <span class="text-ctp-overlay0 font-semibold">Estimated Cost</span>
            <span class="col-span-2 text-ctp-text font-semibold text-ctp-green">${sessionInfo.total_cost.toFixed(4)}</span>
          </div>
        {/if}

        <div class="grid grid-cols-3 gap-2 py-1">
          <span class="text-ctp-overlay0 font-semibold">Status</span>
          <span class="col-span-2 flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full"
                  style="background: {sessionInfo.is_active ? '#65b73b' : '#e95f59'}"></span>
            <span class="text-ctp-text capitalize">{sessionInfo.is_active ? 'Active' : 'Inactive'}</span>
            {#if sessionInfo.status}
              <span class="text-ctp-overlay0">({sessionInfo.status})</span>
            {/if}
          </span>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-ctp-surface0 flex justify-end">
        <button
          class="px-4 py-2 rounded-lg text-xs font-semibold bg-ctp-blue text-white hover:bg-ctp-blue/80 transition-all cursor-pointer shadow-lg shadow-ctp-blue/20"
          onclick={onClose}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
