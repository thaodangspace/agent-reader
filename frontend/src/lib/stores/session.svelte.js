import { writable } from 'svelte/store';

const STORAGE_KEY = 'activeSession';

// Restore from localStorage, default to null
function getInitialActiveSession() {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

export const activeSession = writable(getInitialActiveSession());
export const activeSessionPath = writable(null);
export const sessions = writable([]);
export const unreadSessionIds = writable(new Set());

// Persist activeSession to localStorage
activeSession.subscribe(id => {
  try {
    if (id) {
      localStorage.setItem(STORAGE_KEY, id);
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  } catch {}
});
