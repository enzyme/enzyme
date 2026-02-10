import { useEffect, useRef } from 'react';
import type { MessageListResult } from '@feather/api-client';
import { getCachedIfFresh, getUrls } from '../lib/signedUrlCache';

/** Pre-warms signed URL cache for all attachment IDs found in message pages. */
export function usePrewarmSignedUrls(pages: MessageListResult[] | undefined) {
  const seenRef = useRef(new Set<string>());

  useEffect(() => {
    if (!pages) return;

    const newIds: string[] = [];
    for (const page of pages) {
      for (const msg of page.messages) {
        if (msg.attachments) {
          for (const att of msg.attachments) {
            // Skip if already in-flight, but re-fetch if expired
            if (!seenRef.current.has(att.id) || !getCachedIfFresh(att.id)) {
              seenRef.current.add(att.id);
              newIds.push(att.id);
            }
          }
        }
      }
    }

    if (newIds.length > 0) {
      getUrls(newIds).catch(() => {
        // fire-and-forget — errors will be handled when images render
      });
    }
  }, [pages]);
}
