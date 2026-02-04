import type { User, Workspace, WorkspaceSummary, Channel, Message, MessageWithUser } from '@feather/api-client';

let counter = 0;
function nextId(): string {
  counter++;
  return `test-id-${counter.toString().padStart(6, '0')}`;
}

export function createMockUser(overrides: Partial<User> = {}): User {
  const id = overrides.id ?? nextId();
  return {
    id,
    email: `user-${id}@example.com`,
    display_name: `Test User ${id}`,
    status: '',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export function createMockWorkspace(overrides: Partial<Workspace> = {}): Workspace {
  const id = overrides.id ?? nextId();
  return {
    id,
    name: `Test Workspace ${id}`,
    settings: '{}',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export function createMockWorkspaceSummary(overrides: Partial<WorkspaceSummary> = {}): WorkspaceSummary {
  const id = overrides.id ?? nextId();
  return {
    id,
    name: `Test Workspace ${id}`,
    role: 'member',
    ...overrides,
  };
}

export function createMockChannel(overrides: Partial<Channel> = {}): Channel {
  const id = overrides.id ?? nextId();
  return {
    id,
    workspace_id: overrides.workspace_id ?? nextId(),
    name: `channel-${id}`,
    type: 'public',
    is_default: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export function createMockMessage(overrides: Partial<Message> = {}): Message {
  const id = overrides.id ?? nextId();
  return {
    id,
    channel_id: overrides.channel_id ?? nextId(),
    content: `Test message ${id}`,
    reply_count: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export function createMockMessageWithUser(overrides: Partial<MessageWithUser> = {}): MessageWithUser {
  const message = createMockMessage(overrides);
  return {
    ...message,
    user_display_name: overrides.user_display_name ?? 'Test User',
    user_avatar_url: overrides.user_avatar_url,
    reactions: overrides.reactions ?? [],
    attachments: overrides.attachments ?? [],
    ...overrides,
  };
}

// Pre-built fixtures for common scenarios
export const fixtures = {
  user: createMockUser({ id: 'user-1', email: 'test@example.com', display_name: 'Test User' }),
  workspace: createMockWorkspace({ id: 'ws-1', name: 'Test Workspace' }),
  workspaceSummary: createMockWorkspaceSummary({ id: 'ws-1', name: 'Test Workspace', role: 'owner' }),
  channel: createMockChannel({ id: 'ch-1', workspace_id: 'ws-1', name: 'general' }),
  message: createMockMessage({ id: 'msg-1', channel_id: 'ch-1', user_id: 'user-1', content: 'Hello, world!' }),
};
