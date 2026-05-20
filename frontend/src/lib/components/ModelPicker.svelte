<script>
  import { getAvailableModels, setModel, cycleModel } from '$lib/api/rpc.js';
  import { availableModels, setModelsForSession, clearModelsForSession } from '$lib/stores/models.svelte.js';
  import { rpcRunning, isRpcRunning } from '$lib/stores/rpc.svelte.js';
  import { ChevronDown, Check, RefreshCw } from '@lucide/svelte';

  let { sessionId, currentModel } = $props();

  let showDropdown = $state(false);
  let models = $state([]);
  let loading = $state(false);
  let error = $state('');
  let switching = $state(false);
  let btnRef = $state(null);
  let dropdownEl = $state(null);

  // Dropdown position
  let dropdownTop = $state(0);
  let dropdownRight = $state(0);

  let rpcActive = $derived(isRpcRunning(sessionId));

  async function fetchModels() {
    if (!rpcActive) return;

    let cached;
    availableModels.subscribe(v => { cached = v; })();
    const cachedModels = cached?.get(sessionId);
    if (cachedModels && cachedModels.length > 0) {
      models = cachedModels;
      return;
    }

    loading = true;
    error = '';
    try {
      const resp = await getAvailableModels(sessionId);
      if (resp.success && resp.data?.models) {
        models = resp.data.models;
        setModelsForSession(sessionId, models);
      } else {
        error = resp.error || 'Failed to fetch models';
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function positionDropdown() {
    if (!btnRef || !dropdownEl) return;
    const rect = btnRef.getBoundingClientRect();
    dropdownTop = rect.bottom + 6;
    dropdownRight = window.innerWidth - rect.right;
  }

  // Position dropdown when it becomes visible
  $effect(() => {
    if (showDropdown) {
      // Wait for DOM to update
      requestAnimationFrame(() => {
        positionDropdown();
      });
    }
  });

  // Reposition on scroll/resize
  function reposition() {
    if (showDropdown) positionDropdown();
  }

  async function selectModel(m) {
    if (!rpcActive || switching) return;
    switching = true;
    error = '';
    try {
      const resp = await setModel(sessionId, m.provider, m.id);
      if (resp.success) {
        closeDropdown();
        clearModelsForSession(sessionId);
      } else {
        error = resp.error || 'Failed to switch model';
      }
    } catch (e) {
      error = e.message;
    } finally {
      switching = false;
    }
  }

  async function handleCycle() {
    if (!rpcActive || switching) return;
    switching = true;
    error = '';
    try {
      const resp = await cycleModel(sessionId);
      if (resp.success) {
        closeDropdown();
        clearModelsForSession(sessionId);
      } else {
        error = resp.error || 'No other model available';
      }
    } catch (e) {
      error = e.message;
    } finally {
      switching = false;
    }
  }

  function closeDropdown() {
    showDropdown = false;
  }

  function toggleDropdown(e) {
    e.preventDefault();
    e.stopPropagation();
    if (showDropdown) {
      closeDropdown();
    } else {
      showDropdown = true;
      if (rpcActive && models.length === 0) {
        fetchModels();
      }
    }
  }

  function escapeHTML(str) {
    if (typeof str !== 'string') str = str == null ? '' : String(str);
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function modelLabel(m) {
    return m.name || m.id;
  }

  function modelSublabel(m) {
    const parts = [];
    if (m.provider) parts.push(m.provider);
    if (m.contextWindow) parts.push(`${Math.round(m.contextWindow / 1000)}k ctx`);
    if (m.reasoning) parts.push('thinking');
    return parts.join(' · ');
  }

  function isCurrentModel(m) {
    if (!currentModel) return false;
    return m.id === currentModel || m.id === currentModel.split('.').pop();
  }

  function providerIcon(provider) {
    switch (provider) {
      case 'anthropic': return '🅰';
      case 'openai': return '🅾';
      case 'google': return '🇬';
      case 'bedrock': return '🔷';
      default: return '🤖';
    }
  }
</script>

<!-- Overlay backdrop -->
{#if showDropdown}
  <div
    class="fixed inset-0 z-[99]"
    onclick={closeDropdown}
    onscroll={reposition}
  ></div>
{/if}

<button
  bind:this={btnRef}
  class="text-[11px] px-2 py-0.5 rounded-full whitespace-nowrap cursor-pointer transition-colors flex items-center gap-1 {rpcActive ? 'hover:bg-ctp-blue/30' : 'hover:bg-ctp-surface0/30'}"
  style="background:color-mix(in srgb, #135ce0 20%, transparent); color:#135ce0"
  title={rpcActive ? 'Click to change model' : 'Send a message first to start RPC, then click to change model'}
  onclick={toggleDropdown}
>
  {#if switching}
    <span class="w-3 h-3 border border-ctp-blue border-t-transparent rounded-full animate-spin"></span>
  {:else}
    {currentModel ? escapeHTML(currentModel) : 'no model'}
    <ChevronDown class="w-3 h-3" />
  {/if}
</button>
 
{#if showDropdown}
  <!-- Dropdown panel — fixed position to avoid clipping -->
  <div
    bind:this={dropdownEl}
    class="fixed z-[100] w-80 bg-ctp-base border border-ctp-surface0 rounded-xl shadow-2xl overflow-hidden animate-fadeIn"
    style="top: {dropdownTop}px; right: {dropdownRight}px;"
  >
    <!-- Header -->
    <div class="px-3 py-2 border-b border-ctp-surface0/50 flex items-center justify-between">
      <span class="text-[11px] font-semibold text-ctp-overlay0">Switch Model</span>
      {#if models.length > 1}
        <button
          class="text-[11px] text-ctp-blue hover:text-ctp-blue/80 cursor-pointer px-2 py-0.5 rounded hover:bg-ctp-blue/10 transition-colors inline-flex items-center gap-1"
          disabled={!rpcActive || switching}
          onclick={handleCycle}
          title="Cycle to next model"
        >
          {#if switching}
            <span>...</span>
          {:else}
            <RefreshCw size={11} />
            <span>Cycle</span>
          {/if}
        </button>
      {/if}
    </div>
 
    <!-- RPC not running warning -->
    {#if !rpcActive}
      <div class="px-4 py-4 text-center">
        <div class="text-[11px] text-ctp-overlay0 mb-2">
          RPC is not running for this session.
        </div>
        <div class="text-[10px] text-ctp-overlay1">
          Type a message in the chat box to start RPC, then come back to switch models.
        </div>
      </div>
    {:else if loading}
      <div class="px-4 py-6 text-center text-[11px] text-ctp-overlay0">
        <div class="w-4 h-4 border-2 border-ctp-blue border-t-transparent rounded-full animate-spin mx-auto mb-2"></div>
        Loading models...
      </div>
    {:else if error}
      <div class="px-4 py-3 text-[11px] text-ctp-red bg-ctp-red/5">
        {escapeHTML(error)}
      </div>
    {:else}
      <div class="max-h-72 overflow-y-auto py-1">
        {#each models as m}
          <button
            class="w-full px-3 py-2 text-left flex items-center gap-2.5 transition-colors hover:bg-ctp-surface0/70 cursor-pointer {isCurrentModel(m) ? 'bg-ctp-surface0/40' : ''}"
            disabled={switching || isCurrentModel(m)}
            onclick={() => selectModel(m)}
          >
            <span class="w-4 shrink-0 text-center text-xs flex items-center justify-center">
              {#if isCurrentModel(m)}
                <Check class="w-3 h-3 text-ctp-green" />
              {:else}
                <span class="text-ctp-overlay0 opacity-30">○</span>
              {/if}
            </span>

            <span class="text-sm shrink-0" title={m.provider}>
              {providerIcon(m.provider)}
            </span>

            <div class="flex-1 min-w-0">
              <div class="text-xs font-medium text-ctp-text truncate">
                {escapeHTML(modelLabel(m))}
              </div>
              {#if modelSublabel(m)}
                <div class="text-[10px] text-ctp-overlay0 truncate">{escapeHTML(modelSublabel(m))}</div>
              {/if}
            </div>

            {#if m.contextWindow}
              <span class="text-[9px] px-1.5 py-0.5 rounded shrink-0"
                    style="background:color-mix(in srgb, #1b7c83 15%, transparent); color:#1b7c83">
                {Math.round(m.contextWindow / 1000)}k
              </span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  </div>
{/if}
