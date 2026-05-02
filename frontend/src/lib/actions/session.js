import { activeSession, activeSessionPath, sessions } from '$lib/stores/session.svelte.js';
import { messages, userScrolledUp, newMessageCount } from '$lib/stores/messages.svelte.js';
import { sidebarOpen } from '$lib/stores/ui.svelte.js';
import { fetchSession, fetchSessions } from '$lib/api/sessions.js';
import { clearSeenEvents } from '$lib/utils/events.js';
import { ws } from '$lib/stores/ws.svelte.js';
import { stopRPC } from '$lib/api/rpc.js';
import { isRpcRunning, setRpcRunning } from '$lib/stores/rpc.svelte.js';
import { tick } from 'svelte';

export async function selectSession(id) {
  // Close sidebar on mobile
  if (window.innerWidth <= 768) {
    sidebarOpen.set(false);
  }

  // NOTE: We intentionally do NOT stop the RPC when switching sessions.
  // RPC sessions should keep running so users can switch back without restart delays.

  // Clear chat BEFORE subscribing so replayed events aren't wiped out
  clearSeenEvents();
  messages.set([]);
  userScrolledUp.set(false);
  newMessageCount.set(0);

  activeSession.set(id);

  // Flush DOM updates so container is empty before replay starts
  await tick();

  // Subscribe to the session via WS
  let socket = null;
  ws.subscribe(s => { socket = s; })();
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: 'subscribe', session_id: id }));
  }

  // Fetch session info
  let sessionInfo = null;
  try {
    sessionInfo = await fetchSession(id);
    activeSessionPath.set(sessionInfo.file);
  } catch {}
}

export async function quitSession() {
  let currentActive = null;
  activeSession.subscribe(v => { currentActive = v; })();
  if (!currentActive) return;
  if (!confirm('Quit this session? This will stop the RPC process.')) return;

  if (isRpcRunning(currentActive)) {
    try { await stopRPC(currentActive); } catch {}
    setRpcRunning(currentActive, false);
  }

  activeSession.set(null);
  activeSessionPath.set(null);
  clearSeenEvents();
  messages.set([]);
}

export async function refreshSessions() {
  try {
    const list = await fetchSessions();
    sessions.set(list);
  } catch (e) {
    console.error('Failed to refresh sessions:', e);
  }
}
