let activeChannelId: string | null = null;

export function setActiveChannelId(id: string | null) {
  activeChannelId = id;
}

export function getActiveChannelId(): string | null {
  return activeChannelId;
}
