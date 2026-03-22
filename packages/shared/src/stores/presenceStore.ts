import { useSyncExternalStore } from 'react';
import type { PresenceStatus, TypingEventData } from '@enzyme/api-client';

interface TypingUser {
  userId: string;
  displayName: string;
  expiresAt: number;
}

const TYPING_TIMEOUT = 5000; // 5 seconds

// Module-level state
let typingUsers = new Map<string, TypingUser[]>();
let userPresence = new Map<string, PresenceStatus>();
const listeners = new Set<() => void>();

// Notify all subscribers
function emitChange() {
  listeners.forEach((listener) => listener());
}

// Subscribe for useSyncExternalStore
function subscribe(listener: () => void) {
  listeners.add(listener);
  startCleanupTimer();
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) stopCleanupTimer();
  };
}

// Actions
export function addTypingUser(channelId: string, data: TypingEventData) {
  const newTyping = new Map(typingUsers);
  const channelTypers = newTyping.get(channelId) || [];

  // Remove existing entry for this user
  const filtered = channelTypers.filter((t) => t.userId !== data.user_id);

  // Add new entry with expiration
  filtered.push({
    userId: data.user_id,
    displayName: data.user_display_name || 'Someone',
    expiresAt: Date.now() + TYPING_TIMEOUT,
  });

  newTyping.set(channelId, filtered);
  typingUsers = newTyping;
  emitChange();
}

export function removeTypingUser(channelId: string, userId: string) {
  const newTyping = new Map(typingUsers);
  const channelTypers = newTyping.get(channelId) || [];
  const filtered = channelTypers.filter((t) => t.userId !== userId);

  if (filtered.length === 0) {
    newTyping.delete(channelId);
  } else {
    newTyping.set(channelId, filtered);
  }

  typingUsers = newTyping;
  emitChange();
}

export function setUserPresence(userId: string, status: PresenceStatus) {
  const newPresence = new Map(userPresence);
  newPresence.set(userId, status);
  userPresence = newPresence;
  emitChange();
}

export function setMultipleUserPresence(entries: Array<[string, PresenceStatus]>) {
  const newPresence = new Map(userPresence);
  for (const [userId, status] of entries) {
    newPresence.set(userId, status);
  }
  userPresence = newPresence;
  emitChange();
}

export function clearPresence() {
  userPresence = new Map();
  typingUsers = new Map();
  emitChange();
}

function cleanupExpiredTyping() {
  const now = Date.now();
  let changed = false;
  const newTyping = new Map<string, TypingUser[]>();

  typingUsers.forEach((typers, channelId) => {
    const active = typers.filter((t) => t.expiresAt > now);
    if (active.length !== typers.length) {
      changed = true;
    }
    if (active.length > 0) {
      newTyping.set(channelId, active);
    }
  });

  if (changed) {
    typingUsers = newTyping;
    emitChange();
  }
}

// Lazy cleanup timer — starts on first subscription, stops when all unsubscribe
let cleanupTimer: ReturnType<typeof setInterval> | null = null;

function startCleanupTimer() {
  if (!cleanupTimer) {
    cleanupTimer = setInterval(cleanupExpiredTyping, 1000);
  }
}

function stopCleanupTimer() {
  if (cleanupTimer) {
    clearInterval(cleanupTimer);
    cleanupTimer = null;
  }
}

// Hooks
const EMPTY_TYPERS: TypingUser[] = [];

export function useTypingUsers(channelId: string): TypingUser[] {
  return useSyncExternalStore(
    subscribe,
    () => typingUsers.get(channelId) || EMPTY_TYPERS,
    () => typingUsers.get(channelId) || EMPTY_TYPERS,
  );
}

export function useUserPresence(userId: string): PresenceStatus | undefined {
  return useSyncExternalStore(
    subscribe,
    () => userPresence.get(userId),
    () => userPresence.get(userId),
  );
}
