import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

// Hoist mocks
const mockWorkspacesApi = vi.hoisted(() => ({
  get: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  listMembers: vi.fn(),
  removeMember: vi.fn(),
  updateMemberRole: vi.fn(),
  createInvite: vi.fn(),
  acceptInvite: vi.fn(),
  uploadIcon: vi.fn(),
  deleteIcon: vi.fn(),
  startTyping: vi.fn(),
  stopTyping: vi.fn(),
}));

vi.mock('../api/workspaces', () => ({
  workspacesApi: mockWorkspacesApi,
}));

import {
  useWorkspace,
  useWorkspaceMembers,
  useCreateWorkspace,
  useUpdateMemberRole,
  useRemoveMember,
  useCreateInvite,
  useAcceptInvite,
  useUploadWorkspaceIcon,
  useDeleteWorkspaceIcon,
} from './useWorkspaces';

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient?: QueryClient) {
  const client = queryClient ?? createTestQueryClient();
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe('useWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches workspace by ID', async () => {
    const workspace = { id: 'ws-1', slug: 'test', name: 'Test Workspace' };
    mockWorkspacesApi.get.mockResolvedValue({ workspace });

    const { result } = renderHook(() => useWorkspace('ws-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.workspace).toEqual(workspace);
    expect(mockWorkspacesApi.get).toHaveBeenCalledWith('ws-1');
  });

  it('does not fetch when workspaceId is undefined', () => {
    const { result } = renderHook(() => useWorkspace(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockWorkspacesApi.get).not.toHaveBeenCalled();
  });
});

describe('useWorkspaceMembers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches workspace members', async () => {
    const members = [
      { user_id: 'user-1', role: 'owner', display_name: 'Owner' },
      { user_id: 'user-2', role: 'member', display_name: 'Member' },
    ];
    mockWorkspacesApi.listMembers.mockResolvedValue({ members });

    const { result } = renderHook(() => useWorkspaceMembers('ws-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.members).toEqual(members);
    expect(mockWorkspacesApi.listMembers).toHaveBeenCalledWith('ws-1');
  });

  it('does not fetch when workspaceId is undefined', () => {
    const { result } = renderHook(() => useWorkspaceMembers(undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockWorkspacesApi.listMembers).not.toHaveBeenCalled();
  });
});

describe('useCreateWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a workspace', async () => {
    const queryClient = createTestQueryClient();
    const workspace = { id: 'ws-new', slug: 'new-ws', name: 'New Workspace' };
    mockWorkspacesApi.create.mockResolvedValue({ workspace });

    const { result } = renderHook(() => useCreateWorkspace(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: 'New Workspace' });
    });

    expect(mockWorkspacesApi.create).toHaveBeenCalledWith({
      name: 'New Workspace',
    });
  });

  it('invalidates auth query on success', async () => {
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    mockWorkspacesApi.create.mockResolvedValue({ workspace: { id: 'ws-1' } });

    const { result } = renderHook(() => useCreateWorkspace(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: 'Test' });
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['auth', 'me'] });
  });
});

describe('useUpdateMemberRole', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('updates member role', async () => {
    const queryClient = createTestQueryClient();
    mockWorkspacesApi.updateMemberRole.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useUpdateMemberRole('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ userId: 'user-1', role: 'admin' });
    });

    expect(mockWorkspacesApi.updateMemberRole).toHaveBeenCalledWith('ws-1', 'user-1', 'admin');
  });

  it('invalidates members query on success', async () => {
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    mockWorkspacesApi.updateMemberRole.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useUpdateMemberRole('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ userId: 'user-1', role: 'member' });
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['workspace', 'ws-1', 'members'] });
  });
});

describe('useRemoveMember', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('removes a member from workspace', async () => {
    const queryClient = createTestQueryClient();
    mockWorkspacesApi.removeMember.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useRemoveMember('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('user-123');
    });

    expect(mockWorkspacesApi.removeMember).toHaveBeenCalledWith('ws-1', 'user-123');
  });
});

describe('useCreateInvite', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates an invite', async () => {
    const queryClient = createTestQueryClient();
    const invite = { id: 'inv-1', code: 'ABC123', role: 'member' };
    mockWorkspacesApi.createInvite.mockResolvedValue({ invite });

    const { result } = renderHook(() => useCreateInvite('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync({
        role: 'member',
        max_uses: 10,
        expires_in_hours: 24,
      });
    });

    expect(mockWorkspacesApi.createInvite).toHaveBeenCalledWith('ws-1', {
      role: 'member',
      max_uses: 10,
      expires_in_hours: 24,
    });
    expect(response).toEqual({ invite });
  });
});

describe('useAcceptInvite', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('accepts an invite', async () => {
    const queryClient = createTestQueryClient();
    const workspace = { id: 'ws-1', name: 'Joined Workspace' };
    mockWorkspacesApi.acceptInvite.mockResolvedValue({ workspace });

    const { result } = renderHook(() => useAcceptInvite(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('ABC123');
    });

    expect(mockWorkspacesApi.acceptInvite).toHaveBeenCalledWith('ABC123');
  });

  it('invalidates auth query on success', async () => {
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    mockWorkspacesApi.acceptInvite.mockResolvedValue({ workspace: {} });

    const { result } = renderHook(() => useAcceptInvite(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync('CODE');
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['auth', 'me'] });
  });
});

describe('useUploadWorkspaceIcon', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('uploads workspace icon', async () => {
    const queryClient = createTestQueryClient();
    mockWorkspacesApi.uploadIcon.mockResolvedValue({ icon_url: 'https://example.com/icon.png' });

    const { result } = renderHook(() => useUploadWorkspaceIcon('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    const file = new File(['test'], 'icon.png', { type: 'image/png' });
    await act(async () => {
      await result.current.mutateAsync(file);
    });

    expect(mockWorkspacesApi.uploadIcon).toHaveBeenCalledWith('ws-1', file);
  });

  it('invalidates workspace and auth queries on success', async () => {
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    mockWorkspacesApi.uploadIcon.mockResolvedValue({ icon_url: 'url' });

    const { result } = renderHook(() => useUploadWorkspaceIcon('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    const file = new File(['test'], 'icon.png', { type: 'image/png' });
    await act(async () => {
      await result.current.mutateAsync(file);
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['workspace', 'ws-1'] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['auth', 'me'] });
  });
});

describe('useDeleteWorkspaceIcon', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('deletes workspace icon', async () => {
    const queryClient = createTestQueryClient();
    mockWorkspacesApi.deleteIcon.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useDeleteWorkspaceIcon('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(mockWorkspacesApi.deleteIcon).toHaveBeenCalledWith('ws-1');
  });

  it('invalidates workspace and auth queries on success', async () => {
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    mockWorkspacesApi.deleteIcon.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useDeleteWorkspaceIcon('ws-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['workspace', 'ws-1'] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['auth', 'me'] });
  });
});
