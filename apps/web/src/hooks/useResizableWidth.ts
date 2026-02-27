import { useState, useCallback, useEffect, useRef } from 'react';

export function useResizableWidth(
  key: string,
  defaultWidth: number,
  min: number,
  max: number,
  /** Which side of the panel the divider sits on. 'right' (default) = divider is to the right of the panel, 'left' = divider is to the left. */
  dividerSide: 'right' | 'left' = 'right',
) {
  const [width, setWidth] = useState(() => {
    try {
      const stored = localStorage.getItem(key);
      if (stored) {
        const n = Number(stored);
        if (Number.isFinite(n)) return Math.max(min, Math.min(max, n));
      }
    } catch {
      // ignore
    }
    return defaultWidth;
  });

  const widthRef = useRef(width);
  useEffect(() => {
    widthRef.current = width;
  }, [width]);

  const startXRef = useRef(0);
  const startWidthRef = useRef(0);
  const cleanupRef = useRef<(() => void) | null>(null);

  // Clean up drag listeners if the component unmounts mid-drag
  useEffect(() => {
    return () => cleanupRef.current?.();
  }, []);

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      const el = e.currentTarget as HTMLElement;
      el.setPointerCapture(e.pointerId);

      startXRef.current = e.clientX;
      startWidthRef.current = widthRef.current;

      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';

      const sign = dividerSide === 'right' ? 1 : -1;
      let rafId = 0;

      const cleanup = () => {
        cancelAnimationFrame(rafId);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        el.removeEventListener('pointermove', onPointerMove);
        el.removeEventListener('pointerup', onPointerUp);
        el.removeEventListener('lostpointercapture', onPointerUp);
        cleanupRef.current = null;
      };

      const onPointerMove = (ev: PointerEvent) => {
        cancelAnimationFrame(rafId);
        rafId = requestAnimationFrame(() => {
          const delta = ev.clientX - startXRef.current;
          setWidth(Math.max(min, Math.min(max, startWidthRef.current + delta * sign)));
        });
      };

      const onPointerUp = () => {
        cleanup();
        try {
          localStorage.setItem(key, String(widthRef.current));
        } catch {
          // ignore
        }
      };

      el.addEventListener('pointermove', onPointerMove);
      el.addEventListener('pointerup', onPointerUp);
      el.addEventListener('lostpointercapture', onPointerUp);
      cleanupRef.current = cleanup;
    },
    [min, max, key, dividerSide],
  );

  return { width, dividerProps: { onPointerDown } };
}
