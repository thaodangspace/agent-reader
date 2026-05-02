<script>
  import { onMount } from 'svelte';
  import { wsConnected } from '$lib/stores/ws.svelte.js';
  import { activeSession, sessions } from '$lib/stores/session.svelte.js';
  import { sidebarOpen } from '$lib/stores/ui.svelte.js';
  import { quitSession } from '$lib/actions/session.js';
  import { createSession, fetchSessions } from '$lib/api/sessions.js';
  import { selectSession } from '$lib/actions/session.js';

  let sessionInfo = $state(null);
  let creating = $state(false);

  function fetchSessionInfo(id) {
    if (id) {
      fetch(`/api/sessions/${id}`)
        .then(r => r.json())
        .then(data => { sessionInfo = data; })
        .catch(() => { sessionInfo = null; });
    } else {
      sessionInfo = null;
    }
  }

  async function handleNewSession() {
    if (!sessionInfo?.cwd || creating) return;
    creating = true;
    try {
      const data = await createSession(sessionInfo.cwd);
      const list = await fetchSessions();
      sessions.set(list);
      if (data.session_id) {
        setTimeout(() => selectSession(data.session_id), 300);
      }
    } catch (e) {
      console.error('Failed to create session:', e);
    } finally {
      creating = false;
    }
  }

  onMount(() => {
    // Subscribe to active session changes
    const unsub = activeSession.subscribe(id => {
      fetchSessionInfo(id);
    });
    return unsub;
  });

  function escapeHTML(str) {
    if (typeof str !== 'string') str = str == null ? '' : String(str);
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }
</script>

<div class="px-5 py-2.5 border-b border-ctp-surface0 flex items-center gap-3 bg-ctp-mantle flex-wrap">
  <div
    class="w-2 h-2 rounded-full transition-colors duration-300"
    style="background: {$wsConnected ? '#a6e3a1' : '#f38ba8'}"
  ></div>
  <span class="text-sm text-ctp-overlay0 shrink-0">{$wsConnected ? 'Connected' : 'Disconnected'}</span>

  <div class="flex-1 min-w-0 overflow-hidden flex items-center gap-2">
    {#if sessionInfo}
      {#if sessionInfo.project}
        <span
          class="text-[11px] px-2 py-0.5 rounded-full whitespace-nowrap"
          style="background:color-mix(in srgb, #cba6f7 20%, transparent); color:#cba6f7"
        >
          {escapeHTML(sessionInfo.project)}
        </span>
      {/if}
      {#if sessionInfo.cwd}
        <span
          class="text-[10px] px-2 py-0.5 rounded-full whitespace-nowrap text-ctp-overlay0"
          style="background:color-mix(in srgb, #585b70 20%, transparent)"
        >
          {sessionInfo.cwd.length > 50 ? '...' + sessionInfo.cwd.slice(-47) : sessionInfo.cwd}
        </span>
      {/if}
      {#if sessionInfo.line_count}
        <span
          class="text-[10px] px-2 py-0.5 rounded-full whitespace-nowrap text-ctp-overlay0"
          style="background:color-mix(in srgb, #585b70 20%, transparent)"
        >
          {sessionInfo.line_count} lines
        </span>
      {/if}
    {:else if $activeSession}
      <span
        class="text-[11px] px-2 py-0.5 rounded-full whitespace-nowrap"
        style="background:color-mix(in srgb, #cba6f7 20%, transparent); color:#cba6f7"
      >
        {$activeSession.substring(0, 12)}...
      </span>
    {/if}
  </div>

  {#if sessionInfo?.model}
    <span
      class="text-[11px] px-2 py-0.5 rounded-full whitespace-nowrap"
      style="background:color-mix(in srgb, #89b4fa 20%, transparent); color:#89b4fa"
    >
      {escapeHTML(sessionInfo.model)}
    </span>
  {/if}

  {#if sessionInfo?.cwd}
    <button
      class="px-2.5 py-0.5 rounded-md text-[11px] font-semibold bg-ctp-green/20 text-ctp-green hover:bg-ctp-green/30 transition-colors"
      disabled={creating}
      onclick={handleNewSession}
      title="New session in {sessionInfo.cwd}"
    >
      {creating ? '...' : '+ New'}
    </button>
  {/if}

  {#if $activeSession}
    <button
      class="px-3 py-1 rounded-md text-xs font-semibold bg-ctp-red/20 text-ctp-red hover:bg-ctp-red/30 transition-colors"
      onclick={quitSession}
    >
      Quit Session
    </button>
  {/if}

  <!-- Mobile hamburger -->
  <button
    class="md:hidden absolute top-2.5 left-2.5 z-30 p-1.5 rounded-md bg-ctp-surface0 text-ctp-text hover:bg-ctp-surface1"
    onclick={() => sidebarOpen.update(v => !v)}
  >
    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
    </svg>
  </button>
</div>
