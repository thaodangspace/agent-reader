<script>
  import ToolResultBlock from './ToolResultBlock.svelte';

  let { results } = $props();
  let collapsed = $state(false);
</script>

<div class="flex flex-col items-start animate-fadeIn w-full">
  <div class="w-full max-w-[85%] rounded-xl overflow-hidden border border-ctp-crust"
       style="background:color-mix(in srgb, #135ce0 4%, #ffffff)">
    <!-- Header -->
    <button
      class="w-full flex items-center gap-2 px-3 py-2 text-xs cursor-pointer"
      onclick={() => collapsed = !collapsed}
    >
      <span
        class="transition-transform duration-200 text-[10px]"
        style="transform: {collapsed ? '' : 'rotate(90deg)'}"
      >▶</span>
      <span>📎</span>
      <span class="font-semibold" style="color:#135ce0">Tool Results</span>
      <span class="text-ctp-overlay0 text-[10px] ml-auto">{results.length} results</span>
    </button>

    <!-- Individual results (shown when expanded) -->
    {#if !collapsed}
      <div>
        {#each results as result, i}
          {#if i > 0}
            <div class="border-t border-ctp-surface0/30"></div>
          {/if}
          <ToolResultBlock msg={result} standalone={false} />
        {/each}
      </div>
    {/if}
  </div>
</div>
