import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const mockFilesApi = vi.hoisted(() => ({
  upload: vi.fn(),
  signUrl: vi.fn(),
  signUrls: vi.fn(),
}));

vi.mock('@enzyme/api-client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@enzyme/api-client')>();
  return { ...original, filesApi: mockFilesApi };
});

import { useUploadFile } from './useFiles';

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

describe('useUploadFile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls filesApi.upload with channelId and file', async () => {
    const queryClient = createTestQueryClient();
    const uploadResult = {
      file: {
        id: 'file-1',
        filename: 'test.pdf',
        size: 1024,
        content_type: 'application/pdf',
      },
    };
    mockFilesApi.upload.mockResolvedValue(uploadResult);

    const file = new File(['test content'], 'test.pdf', { type: 'application/pdf' });

    const { result } = renderHook(() => useUploadFile('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(file);
    });

    expect(mockFilesApi.upload).toHaveBeenCalledWith('channel-1', file);
  });

  it('returns upload result', async () => {
    const queryClient = createTestQueryClient();
    const uploadResult = {
      file: {
        id: 'file-2',
        filename: 'image.jpg',
        size: 2048,
        content_type: 'image/jpeg',
      },
    };
    mockFilesApi.upload.mockResolvedValue(uploadResult);

    const file = new File(['image data'], 'image.jpg', { type: 'image/jpeg' });

    const { result } = renderHook(() => useUploadFile('channel-1'), {
      wrapper: createWrapper(queryClient),
    });

    let response;
    await act(async () => {
      response = await result.current.mutateAsync(file);
    });

    expect(response).toEqual(uploadResult);
  });

  it('handles different channel IDs', async () => {
    const queryClient = createTestQueryClient();
    mockFilesApi.upload.mockResolvedValue({
      file: { id: 'file-3', filename: 'doc.txt', size: 100, content_type: 'text/plain' },
    });

    const file = new File(['content'], 'doc.txt', { type: 'text/plain' });

    const { result } = renderHook(() => useUploadFile('another-channel'), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(file);
    });

    expect(mockFilesApi.upload).toHaveBeenCalledWith('another-channel', file);
  });
});
