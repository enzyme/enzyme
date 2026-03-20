import { useSyncExternalStore } from 'react';

// Module-level state
let editingMessageId: string | null = null;
const listeners = new Set<() => void>();

function emitChange() {
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot() {
  return editingMessageId;
}

export function setEditingMessageId(id: string) {
  editingMessageId = id;
  emitChange();
}

export function clearEditingMessageId() {
  editingMessageId = null;
  emitChange();
}

/** Returns the raw editing message ID. Prefer useIsEditingMessage for components. */
export function useEditingMessageId(): string | null {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

/**
 * Selector hook that only re-renders when this specific message's editing
 * state changes (boolean comparison via Object.is), avoiding the re-render
 * blast radius of subscribing to the raw ID.
 */
export function useIsEditingMessage(messageId: string): boolean {
  return useSyncExternalStore(
    subscribe,
    () => editingMessageId === messageId,
    () => false,
  );
}
