export async function fetchSessions(sortBy = 'last_updated', groupBy = false) {
  const res = await fetch(`/api/sessions?page=1&sort=${sortBy}&group_by=${groupBy ? 'project' : ''}`);
  if (!res.ok) throw new Error('Failed to fetch sessions');
  const data = await res.json();
  return data.sessions;
}

export async function fetchSession(id) {
  const res = await fetch(`/api/sessions/${id}`);
  if (!res.ok) throw new Error('Session not found');
  return res.json();
}

export async function createSession(cwd) {
  const res = await fetch('/api/sessions/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cwd }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function fetchUnreadIds() {
  const res = await fetch('/api/sessions/unread');
  if (!res.ok) throw new Error('Failed to fetch unread IDs');
  const data = await res.json();
  return new Set(data.unread_ids || []);
}

export async function markSessionRead(id) {
  const res = await fetch(`/api/sessions/${id}/mark-read`, {
    method: 'POST',
  });
  if (!res.ok) throw new Error('Failed to mark session as read');
  return res.json();
}
