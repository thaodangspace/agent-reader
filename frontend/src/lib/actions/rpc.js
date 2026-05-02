import { activeSession } from '$lib/stores/session.svelte.js';
import { isStreaming, rpcAutoStarting, warnedSessions, setRpcRunning, isRpcRunning } from '$lib/stores/rpc.svelte.js';
import { messages } from '$lib/stores/messages.svelte.js';
import { startRPC, stopRPC, sendRPC } from '$lib/api/rpc.js';
import { addSystemMessage } from '$lib/utils/events.js';

export async function toggleRPC() {
  let currentActive = null;
  activeSession.subscribe(v => { currentActive = v; })();
  if (!currentActive) return;

  const currentRpc = isRpcRunning(currentActive);

  if (currentRpc) {
    try {
      await stopRPC(currentActive);
      setRpcRunning(currentActive, false);
    } catch (e) {
      console.error('RPC stop error:', e);
    }
  } else {
    try {
      await startRPC(currentActive);
      setRpcRunning(currentActive, true);
    } catch (e) {
      addSystemMessage('Failed to start RPC: ' + e.message);
    }
  }
}

export async function abortRPC() {
  let currentActive = null;
  activeSession.subscribe(v => { currentActive = v; })();
  if (!currentActive || !isRpcRunning(currentActive)) return;
  try {
    await sendRPC(currentActive, { type: 'abort' });
  } catch (e) {
    console.error('Abort error:', e);
  }
}

export async function sendMessage(text) {
  if (!text) return;

  let currentActive = null;
  activeSession.subscribe(v => { currentActive = v; })();
  if (!currentActive) {
    addSystemMessage('No session selected');
    return;
  }

  const currentRpc = isRpcRunning(currentActive);

  // Auto-start RPC if not running
  if (!currentRpc) {
    let warnedSet = new Set();
    warnedSessions.subscribe(v => { warnedSet = new Set(v); })();

    // Warn user on first send in this session
    if (!warnedSet.has(currentActive)) {
      warnedSet.add(currentActive);
      warnedSessions.set(warnedSet);
      addSystemMessage('⚡ Auto-starting RPC for this session...');
    }

    rpcAutoStarting.set(true);
    try {
      await startRPC(currentActive);
      setRpcRunning(currentActive, true);
    } catch (e) {
      rpcAutoStarting.set(false);
      addSystemMessage('Failed to start RPC: ' + e.message);
      return;
    }
    rpcAutoStarting.set(false);
  }

  let currentStreaming = false;
  isStreaming.subscribe(v => { currentStreaming = v; })();

  const cmd = { type: 'prompt', message: text };
  if (currentStreaming) cmd.streamingBehavior = 'steer';

  try {
    await sendRPC(currentActive, cmd);
  } catch (e) {
    addSystemMessage('Failed to send: ' + e.message);
  }
}
