# Group By Project Toggle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a toggle in the sidebar to switch between flat session list and sessions grouped by project.

**Architecture:** Purely frontend — a new `groupByProject` store (persisted to localStorage) controls whether `Sidebar.svelte` renders sessions flat or grouped. Grouping is done via an inline `{#each}` over grouped data. No backend changes.

**Tech Stack:** Svelte 5 (runes), Vite, Tailwind CSS

---

### Task 1: Add `groupByProject` Store

**Files:**
- Modify: `frontend/src/lib/stores/ui.svelte.js`

- [ ] **Step 1: Add the `groupByProject` store with localStorage persistence**

```js
import { writable } from 'svelte/store';

export const sidebarOpen = writable(false);
export const newSessionModalOpen = writable(false);

// Restore from localStorage, default to false
function getInitialGroupByProject() {
  try {
    return localStorage.getItem('groupByProject') === 'true';
  } catch {
    return false;
  }
}

export const groupByProject = writable(getInitialGroupByProject());

// Persist to localStorage
groupByProject.subscribe(value => {
  try {
    localStorage.setItem('groupByProject', String(value));
  } catch {}
});
```

- [ ] **Step 2: Verify the store compiles**

Run: `cd frontend && npx vite build 2>&1 | head -20`
Expected: No errors (Svelte stores are trivial to compile)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/stores/ui.svelte.js
git commit -m "feat: add groupByProject store with localStorage persistence"
```

### Task 2: Add Toggle Button to Sidebar Header

**Files:**
- Modify: `frontend/src/lib/components/Sidebar.svelte`

- [ ] **Step 1: Add the toggle button between the title and the `＋` button**

In the sidebar header div (line ~12), add the toggle button:

```svelte
<div class="p-4 border-b border-ctp-surface0 text-sm font-semibold text-ctp-blue flex items-center justify-between">
  <span>⚡ Sessions</span>
  <div class="flex items-center gap-2">
    <button
      class="text-ctp-green hover:text-ctp-teal text-xs font-bold"
      onclick={() => groupByProject.update(v => !v)}
      title={groupByProject ? "Switch to flat list" : "Group by project"}
    >{groupByProject ? '📁' : '≡'}</button>
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
```

Add the import at the top:

```svelte
import { groupByProject } from '$lib/stores/ui.svelte.js';
```

- [ ] **Step 2: Verify it compiles**

Run: `cd frontend && npx vite build 2>&1 | head -20`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/components/Sidebar.svelte
git commit -m "feat: add group-by-project toggle button in sidebar header"
```

### Task 3: Implement Grouped Rendering in Sidebar

**Files:**
- Modify: `frontend/src/lib/components/Sidebar.svelte`

- [ ] **Step 1: Add grouped rendering logic**

Replace the session list rendering section. The current flat `{#each}` block needs to become conditional:

```svelte
<div class="flex-1 overflow-y-auto">
  {#if $sessions.length === 0}
    <div class="flex items-center justify-center h-full text-ctp-overlay0 text-sm">
      No sessions yet
    </div>
  {:else if $groupByProject}
    <!-- Grouped by project -->
    {#each groupedSessions as { project, sessions: projectSessions } (project)}
      <div class="project-group">
        <button
          class="w-full px-4 py-2 text-xs font-semibold text-ctp-subtext0 flex items-center justify-between hover:bg-ctp-surface1 cursor-pointer border-b border-ctp-surface0"
          onclick={() => toggleProjectGroup(project)}
        >
          <span>{project} ({projectSessions.length})</span>
          <span>{expandedProjects[project] ? '▼' : '▶'}</span>
        </button>
        {#if expandedProjects[project]}
          {#each projectSessions as session (session.id)}
            <div
              class="session-item px-4 py-2.5 border-b border-ctp-surface0 cursor-pointer transition-colors duration-150 hover:bg-ctp-surface1 {$activeSession === session.id ? 'bg-ctp-surface0 border-l-[3px] border-ctp-blue' : ''}"
              onclick={() => selectSession(session.id)}
            >
              <div class="flex items-center justify-between">
                <div class="text-xs text-ctp-text">{session.project}</div>
                {#if session.last_message_time}
                  <div class="text-[10px] text-ctp-overlay0">{session.last_message_time}</div>
                {/if}
              </div>
              <div class="text-[11px] text-ctp-overlay1 break-all">{session.id}</div>
              <div class="text-[10px] text-ctp-overlay0 mt-0.5">{session.cwd}</div>
              {#if session.model}
                <div class="text-[10px] text-ctp-blue mt-0.5">{session.model}</div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    {/each}
  {:else}
    <!-- Flat list (existing behavior) -->
    {#each $sessions as session (session.id)}
      ...existing session item rendering...
    {/each}
  {/if}
</div>
```

- [ ] **Step 2: Add the `groupedSessions` computed value and expanded state**

Add a derived store or inline computed value for grouping:

```svelte
<script>
  // ... existing imports ...
  import { groupByProject } from '$lib/stores/ui.svelte.js';

  // Expanded project groups state
  let expandedProjects = $state({});

  // Computed grouped sessions
  let groupedSessions = $derived.by(() => {
    const sessions = $sessions;
    const groups = {};
    for (const session of sessions) {
      if (!groups[session.project]) {
        groups[session.project] = [];
      }
      groups[session.project].push(session);
    }
    // Sort groups alphabetically, sessions within groups by timestamp (newest first)
    return Object.keys(groups)
      .sort()
      .map(project => ({
        project,
        sessions: groups[project].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
      }));
  });

  // Auto-expand group containing active session
  $effect(() => {
    const activeId = $activeSession;
    if (activeId) {
      for (const group of groupedSessions) {
        if (group.sessions.some(s => s.id === activeId)) {
          expandedProjects[group.project] = true;
          break;
        }
      }
    }
  });

  function toggleProjectGroup(project) {
    expandedProjects[project] = !expandedProjects[project];
  }
</script>
```

- [ ] **Step 3: Verify it compiles**

Run: `cd frontend && npx vite build 2>&1 | head -20`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/components/Sidebar.svelte
git commit -m "feat: implement group-by-project rendering in sidebar"
```

### Task 4: Manual Verification

- [ ] **Step 1: Build and start the server**

```bash
cd frontend && npm run build
cd .. && go build -o bin/server ./cmd/server
./bin/server
```

- [ ] **Step 2: Verify in browser**

1. Open `http://localhost:8080`
2. Click toggle button — sessions should group by project with collapsible headers
3. Click toggle again — should return to flat list
4. Refresh page — toggle state should persist
5. Collapse a group, expand another — verify collapsible behavior
6. Select an active session — its project group should auto-expand

- [ ] **Step 3: Commit**

```bash
git commit --allow-empty -m "chore: verify group-by-project toggle works"
```
