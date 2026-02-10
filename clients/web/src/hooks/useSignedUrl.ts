import { useState, useEffect } from 'react';
import { getCachedIfFresh, getUrl } from '../lib/signedUrlCache';

/** Returns a signed download URL for the given file ID, or null while loading. */
export function useSignedUrl(fileId: string): string | null {
  const [url, setUrl] = useState<string | null>(() => getCachedIfFresh(fileId));
  const [prevFileId, setPrevFileId] = useState(fileId);

  // Reset state synchronously during render when fileId changes (React-recommended pattern)
  if (fileId !== prevFileId) {
    setPrevFileId(fileId);
    setUrl(getCachedIfFresh(fileId));
  }

  useEffect(() => {
    let cancelled = false;

    if (getCachedIfFresh(fileId)) return;

    getUrl(fileId)
      .then((u) => {
        if (!cancelled) setUrl(u);
      })
      .catch(() => {
        // Error will surface when the image fails to load and triggers a retry
      });

    return () => {
      cancelled = true;
    };
  }, [fileId]);

  return url;
}
