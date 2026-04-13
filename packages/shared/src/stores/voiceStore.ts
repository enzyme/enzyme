import { useSyncExternalStore } from 'react';

export interface VoiceParticipantState {
  userId: string;
  muted: boolean;
  deafened: boolean;
  serverMuted: boolean;
}

// Module-level state
let activeChannelId: string | null = null;
let participants = new Map<string, VoiceParticipantState[]>();
let speakingUsers = new Set<string>();
let localMuted = false;
let localDeafened = false;
const listeners = new Set<() => void>();

function emitChange() {
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

// Actions
export function setActiveVoiceChannel(channelId: string | null) {
  activeChannelId = channelId;
  emitChange();
}

export function addVoiceParticipant(channelId: string, userId: string) {
  const newParticipants = new Map(participants);
  const channelParticipants = newParticipants.get(channelId) || [];
  if (!channelParticipants.some((p) => p.userId === userId)) {
    newParticipants.set(channelId, [
      ...channelParticipants,
      { userId, muted: false, deafened: false, serverMuted: false },
    ]);
  }
  participants = newParticipants;
  emitChange();
}

export function removeVoiceParticipant(channelId: string, userId: string) {
  const newParticipants = new Map(participants);
  const channelParticipants = newParticipants.get(channelId) || [];
  const filtered = channelParticipants.filter((p) => p.userId !== userId);
  if (filtered.length === 0) {
    newParticipants.delete(channelId);
  } else {
    newParticipants.set(channelId, filtered);
  }
  participants = newParticipants;
  emitChange();
}

export function setVoiceParticipantMuteState(
  channelId: string,
  userId: string,
  muted: boolean,
  deafened: boolean,
  serverMuted: boolean,
) {
  const newParticipants = new Map(participants);
  const channelParticipants = newParticipants.get(channelId) || [];
  newParticipants.set(
    channelId,
    channelParticipants.map((p) =>
      p.userId === userId ? { ...p, muted, deafened, serverMuted } : p,
    ),
  );
  participants = newParticipants;
  emitChange();
}

export function setVoiceSpeaking(userId: string, speaking: boolean) {
  const newSpeaking = new Set(speakingUsers);
  if (speaking) {
    newSpeaking.add(userId);
  } else {
    newSpeaking.delete(userId);
  }
  speakingUsers = newSpeaking;
  emitChange();
}

export function setLocalMuted(muted: boolean) {
  localMuted = muted;
  emitChange();
}

export function setLocalDeafened(deafened: boolean) {
  localDeafened = deafened;
  if (deafened) localMuted = true;
  emitChange();
}

export function clearVoiceState() {
  activeChannelId = null;
  participants = new Map();
  speakingUsers = new Set();
  localMuted = false;
  localDeafened = false;
  emitChange();
}

// Hooks
const EMPTY_PARTICIPANTS: VoiceParticipantState[] = [];

export function useActiveVoiceChannel(): string | null {
  return useSyncExternalStore(
    subscribe,
    () => activeChannelId,
    () => null,
  );
}

export function useVoiceChannelParticipants(channelId: string): VoiceParticipantState[] {
  return useSyncExternalStore(
    subscribe,
    () => participants.get(channelId) || EMPTY_PARTICIPANTS,
    () => EMPTY_PARTICIPANTS,
  );
}

export function useIsUserSpeaking(userId: string): boolean {
  return useSyncExternalStore(
    subscribe,
    () => speakingUsers.has(userId),
    () => false,
  );
}

export function useLocalMuted(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => localMuted,
    () => false,
  );
}

export function useLocalDeafened(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => localDeafened,
    () => false,
  );
}
