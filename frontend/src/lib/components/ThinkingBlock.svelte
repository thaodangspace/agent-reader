<script>
  import { escapeHTML } from '$lib/utils/markdown.js';
  import { ChevronRight, ChevronDown, Brain } from '@lucide/svelte';

  let { content } = $props();
  let collapsed = $state(true);

  function toggle() {
    collapsed = !collapsed;
  }
</script>

<div
  class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2"
  style="background:color-mix(in srgb, #135ce0 6%, #ffffff)"
>
  <button
    class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer"
    onclick={toggle}
  >
    <span class="flex items-center">
      {#if collapsed}
        <ChevronRight size={12} />
      {:else}
        <ChevronDown size={12} />
      {/if}
    </span>
    <Brain size={14} class="text-ctp-blue" />
    <span class="font-semibold text-ctp-blue">Thinking</span>
    <span class="text-ctp-overlay0 text-[10px] ml-auto">{escapeHTML(content.substring(0, 60))}…</span>
  </button>
  <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
    <div class="p-3 text-xs" style="background:#f6f6f6;">
      <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto text-ctp-blue/70">{content}</pre>
    </div>
  </div>
</div>
