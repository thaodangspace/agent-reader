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

// Restore from localStorage, default to 'last_updated'
function getInitialSortBy() {
  try {
    return localStorage.getItem('sortBy') || 'last_updated';
  } catch {
    return 'last_updated';
  }
}

export const sortBy = writable(getInitialSortBy());

// Persist to localStorage
sortBy.subscribe(value => {
  try {
    localStorage.setItem('sortBy', value);
  } catch {}
});
