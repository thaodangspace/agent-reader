import { writable } from 'svelte/store';

// Per-session RPC status: Map<sessionId, boolean>
export const rpcRunning = writable(new Map());
export const isStreaming = writable(false);
export const rpcAutoStarting = writable(false);
export const warnedSessions = writable(new Set());

// Helper: check if a specific session's RPC is running
export function isRpcRunning(sessionId) {
  let map;
  rpcRunning.subscribe(v => { map = v; })();
  return map.get(sessionId) === true;
}

// Helper: set RPC running status for a session
export function setRpcRunning(sessionId, running) {
  rpcRunning.update(map => {
    const next = new Map(map);
    if (running) {
      next.set(sessionId, true);
    } else {
      next.delete(sessionId);
    }
    return next;
  });
}
