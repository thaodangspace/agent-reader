<script>
  import { toast } from '$lib/stores/toast.svelte.js';
  import { CheckCircle2, AlertCircle, Info } from '@lucide/svelte';
</script>

<div class="fixed bottom-5 right-5 z-[10000] flex flex-col gap-2 max-w-sm pointer-events-none">
  {#each $toast as t (t.id)}
    <div
      class="flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg pointer-events-auto border animate-fadeIn bg-ctp-mantle border-ctp-surface0"
      role="alert"
    >
      {#if t.type === 'success'}
        <CheckCircle2 size={16} class="text-ctp-green shrink-0" />
      {:else if t.type === 'error'}
        <AlertCircle size={16} class="text-ctp-red shrink-0" />
      {:else}
        <Info size={16} class="text-ctp-blue shrink-0" />
      {/if}
      <span class="text-xs text-ctp-text font-mono font-medium">{t.message}</span>
    </div>
  {/each}
</div>
