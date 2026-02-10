import { filesApi } from '../api/files';

interface CacheEntry {
  url: string;
  expiresAt: number; // unix ms
}

const REFRESH_BUFFER_MS = 5 * 60 * 1000; // 5 minutes

const cache = new Map<string, CacheEntry>();
const pendingFetches = new Map<string, Promise<string>>();

function isFresh(entry: CacheEntry): boolean {
  return entry.expiresAt - Date.now() > REFRESH_BUFFER_MS;
}

/** Synchronous check — returns URL if cached and fresh, else null. */
export function getCachedIfFresh(fileId: string): string | null {
  const entry = cache.get(fileId);
  if (entry && isFresh(entry)) return entry.url;
  return null;
}

/** Async — returns cached if fresh, else fetches a signed URL. Deduplicates in-flight requests. */
export async function getUrl(fileId: string): Promise<string> {
  const cached = getCachedIfFresh(fileId);
  if (cached) return cached;

  const pending = pendingFetches.get(fileId);
  if (pending) return pending;

  const promise = filesApi
    .signUrl(fileId)
    .then((signed) => {
      cache.set(fileId, {
        url: signed.url,
        expiresAt: new Date(signed.expires_at).getTime(),
      });
      return signed.url;
    })
    .finally(() => {
      pendingFetches.delete(fileId);
    });

  pendingFetches.set(fileId, promise);
  return promise;
}

/** Batch fetch — filters to stale/missing IDs (skipping in-flight singles), calls batch endpoint, caches all. */
export async function getUrls(fileIds: string[]): Promise<void> {
  const stale = fileIds.filter((id) => !getCachedIfFresh(id) && !pendingFetches.has(id));
  if (stale.length === 0) return;

  const { urls } = await filesApi.signUrls(stale);
  for (const signed of urls) {
    cache.set(signed.file_id, {
      url: signed.url,
      expiresAt: new Date(signed.expires_at).getTime(),
    });
  }
}

/** Remove a cache entry (e.g. after a load error). */
export function invalidate(fileId: string): void {
  cache.delete(fileId);
}
