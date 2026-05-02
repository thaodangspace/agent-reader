export async function startRPC(sessionId) {
  const res = await fetch('/api/rpc/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function stopRPC(sessionId) {
  const res = await fetch('/api/rpc/stop', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function sendRPC(sessionId, command) {
  const res = await fetch('/api/rpc/send', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId, command }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function getRPCStatus() {
  const res = await fetch('/api/rpc/status');
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
