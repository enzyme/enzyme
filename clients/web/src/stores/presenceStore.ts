import { create } from 'zustand';
import type { PresenceStatus, TypingEventData } from '@feather/api-client';

interface TypingUser {
  userId: string;
  displayName: string;
  expiresAt: number;
}

interface PresenceState {
  // User presence: userId -> status
  userPresence: Map<string, PresenceStatus>;

  // Typing indicators: channelId -> array of typing users
  typingUsers: Map<string, TypingUser[]>;

  // Actions
  setUserPresence: (userId: string, status: PresenceStatus) => void;
  addTypingUser: (channelId: string, data: TypingEventData) => void;
  removeTypingUser: (channelId: string, userId: string) => void;
  getTypingUsers: (channelId: string) => TypingUser[];
  cleanupExpiredTyping: () => void;
}

const TYPING_TIMEOUT = 5000; // 5 seconds

export const usePresenceStore = create<PresenceState>((set, get) => ({
  userPresence: new Map(),
  typingUsers: new Map(),

  setUserPresence: (userId, status) =>
    set((state) => {
      const newPresence = new Map(state.userPresence);
      newPresence.set(userId, status);
      return { userPresence: newPresence };
    }),

  addTypingUser: (channelId, data) =>
    set((state) => {
      const newTyping = new Map(state.typingUsers);
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
      return { typingUsers: newTyping };
    }),

  removeTypingUser: (channelId, userId) =>
    set((state) => {
      const newTyping = new Map(state.typingUsers);
      const channelTypers = newTyping.get(channelId) || [];
      const filtered = channelTypers.filter((t) => t.userId !== userId);

      if (filtered.length === 0) {
        newTyping.delete(channelId);
      } else {
        newTyping.set(channelId, filtered);
      }

      return { typingUsers: newTyping };
    }),

  getTypingUsers: (channelId) => {
    const state = get();
    const now = Date.now();
    const typers = state.typingUsers.get(channelId) || [];
    return typers.filter((t) => t.expiresAt > now);
  },

  cleanupExpiredTyping: () =>
    set((state) => {
      const now = Date.now();
      const newTyping = new Map<string, TypingUser[]>();

      state.typingUsers.forEach((typers, channelId) => {
        const active = typers.filter((t) => t.expiresAt > now);
        if (active.length > 0) {
          newTyping.set(channelId, active);
        }
      });

      return { typingUsers: newTyping };
    }),
}));

// Cleanup expired typing indicators every second
setInterval(() => {
  usePresenceStore.getState().cleanupExpiredTyping();
}, 1000);
