import { writable } from 'svelte/store';

const toasts = writable([]);

export const toast = {
  subscribe: toasts.subscribe,
  show(message, type = 'info', duration = 3000) {
    const id = Math.random().toString(36).substring(2, 9);
    toasts.update(list => [...list, { id, message, type }]);
    setTimeout(() => {
      toasts.update(list => list.filter(t => t.id !== id));
    }, duration);
  },
  success(message, duration) {
    this.show(message, 'success', duration);
  },
  error(message, duration) {
    this.show(message, 'error', duration);
  },
  info(message, duration) {
    this.show(message, 'info', duration);
  }
};
