const STORAGE_KEY = 'enzyme_recent_channels';
const MAX_RECENT = 8;

interface RecentStore {
  [workspaceId: string]: string[];
}

function loadStore(): RecentStore {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function saveStore(store: RecentStore): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(store));
}

export function getRecentChannels(workspaceId: string): string[] {
  const store = loadStore();
  return store[workspaceId] ?? [];
}

export function recordChannelVisit(workspaceId: string, channelId: string): void {
  const store = loadStore();
  const list = store[workspaceId] ?? [];
  // Remove if already present, then prepend
  const filtered = list.filter((id) => id !== channelId);
  filtered.unshift(channelId);
  store[workspaceId] = filtered.slice(0, MAX_RECENT);
  saveStore(store);
}
