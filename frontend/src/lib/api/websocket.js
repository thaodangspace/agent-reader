import { ws, wsConnected } from '$lib/stores/ws.svelte.js';
import { activeSession } from '$lib/stores/session.svelte.js';
import { onWSMessage } from '$lib/utils/events.js';

let socket = null;

export function connectWS() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  socket = new WebSocket(`${proto}//${location.host}/ws`);
  ws.set(socket);

  socket.onopen = () => {
    wsConnected.set(true);
    console.log('WS connected');
    // Re-subscribe to active session on reconnect
    const unsub = activeSession.subscribe(id => {
      if (id && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'subscribe', session_id: id }));
      }
    });
  };

  socket.onclose = () => {
    wsConnected.set(false);
    console.log('WS disconnected — reconnecting...');
    setTimeout(connectWS, 3000);
  };

  socket.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === 'event') {
        onWSMessage(msg);
      }
    } catch (e) {
      console.error('WS parse error:', e);
    }
  };
}
