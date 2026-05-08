<script>
  import { sessions, activeSession } from '$lib/stores/session.svelte.js';
  import { sidebarOpen, groupByProject } from '$lib/stores/ui.svelte.js';
  import { selectSession } from '$lib/actions/session.js';

  let { onNewSession } = $props();

  // Expanded project groups state
  let expandedProjects = $state({});

  // Computed grouped sessions: sorted alphabetically by cwd, sessions within sorted by timestamp (newest first)
  let groupedSessions = $derived.by(() => {
    const list = $sessions;
    const groups = {};
    for (const session of list) {
      const key = session.cwd || session.project || 'unknown';
      if (!groups[key]) {
        groups[key] = [];
      }
      groups[key].push(session);
    }
    return Object.keys(groups)
      .sort()
      .map(cwd => ({
        cwd,
        sessions: groups[cwd].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
      }));
  });

  // Auto-expand group containing active session
  $effect(() => {
    const activeId = $activeSession;
    if (activeId) {
      for (const group of groupedSessions) {
        if (group.sessions.some(s => s.id === activeId)) {
          expandedProjects[group.cwd] = true;
          break;
        }
      }
    }
  });

  function toggleProjectGroup(cwd) {
    expandedProjects[cwd] = !expandedProjects[cwd];
  }
</script>

{#snippet sessionItem(session)}
  <div
    class="session-item px-4 py-2.5 border-b border-ctp-surface0 cursor-pointer transition-colors duration-150 hover:bg-ctp-surface1 {$activeSession === session.id ? 'bg-ctp-surface0 border-l-[3px] border-ctp-blue' : ''}"
    onclick={() => selectSession(session.id)}
  >
    <div class="flex items-center justify-between">
      <div class="text-xs text-ctp-text truncate">{session.project}</div>
      {#if session.last_message_time}
        <div class="text-[10px] text-ctp-overlay0">{session.last_message_time}</div>
      {/if}
    </div>
    {#if session.first_user_message}
      <div class="text-[11px] text-ctp-overlay1 truncate" title={session.first_user_message}>{session.first_user_message}</div>
    {:else}
      <div class="text-[11px] text-ctp-overlay1 break-all">{session.id}</div>
    {/if}
    <div class="text-[10px] text-ctp-overlay0 mt-0.5">{session.cwd}</div>
    <div class="flex items-center gap-2 mt-0.5">
      <span class="text-[9px] font-semibold px-1.5 py-0.5 rounded bg-ctp-mauve/20 text-ctp-mauve">{session.agent || 'pi'}</span>
      {#if session.model}
        <span class="text-[10px] text-ctp-blue">{session.model}</span>
      {/if}
    </div>
  </div>
{/snippet}

<div class="w-[280px] h-full bg-ctp-mantle border-r border-ctp-surface0 flex flex-col">
  <div class="p-4 border-b border-ctp-surface0 text-sm font-semibold text-ctp-blue flex items-center justify-between">
    <span>⚡ Sessions</span>
    <div class="flex items-center gap-2">
      <button
        class="text-ctp-green hover:text-ctp-teal text-xs font-bold"
        onclick={() => groupByProject.update(v => !v)}
        title={$groupByProject ? "Switch to flat list" : "Group by project"}
      >{$groupByProject ? '📁' : '≡'}</button>
      <button
        class="text-ctp-green hover:text-ctp-teal text-xs font-bold"
        onclick={onNewSession}
        title="New Session"
      >＋</button>
      <button
        class="md:hidden text-ctp-overlay0 hover:text-ctp-text"
        onclick={() => sidebarOpen.set(false)}
      >✕</button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if $sessions.length === 0}
      <div class="flex items-center justify-center h-full text-ctp-overlay0 text-sm">
        No sessions yet
      </div>
    {:else if $groupByProject}
      <!-- Grouped by cwd -->
      {#each groupedSessions as { cwd, sessions: cwdSessions } (cwd)}
        <div class="project-group">
          <button
            class="w-full px-4 py-2 text-xs font-semibold text-ctp-subtext0 flex items-center justify-between hover:bg-ctp-surface1 cursor-pointer border-b border-ctp-surface0"
            onclick={() => toggleProjectGroup(cwd)}
          >
            <span class="truncate">{cwd} ({cwdSessions.length})</span>
            <span class="ml-2 flex-shrink-0">{expandedProjects[cwd] ? '▼' : '▶'}</span>
          </button>
          {#if expandedProjects[cwd]}
            {#each cwdSessions as session (session.id)}
              {@render sessionItem(session)}
            {/each}
          {/if}
        </div>
      {/each}
    {:else}
      <!-- Flat list -->
      {#each $sessions as session (session.id)}
        {@render sessionItem(session)}
      {/each}
    {/if}
  </div>
</div>
