import { useState, useCallback } from 'react';
import type { Attachment } from '@enzyme/api-client';

interface ImageViewerState {
  images: Attachment[];
  index: number;
}

export function useImageViewer() {
  const [viewer, setViewer] = useState<ImageViewerState | null>(null);

  const openViewer = useCallback((images: Attachment[], index: number) => {
    setViewer({ images, index });
  }, []);

  const closeViewer = useCallback(() => {
    setViewer(null);
  }, []);

  return { viewer, openViewer, closeViewer };
}
