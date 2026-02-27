import { useState, useCallback, useRef } from 'react';

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

  const startXRef = useRef(0);
  const startWidthRef = useRef(0);

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();
      startXRef.current = e.clientX;
      startWidthRef.current = width;

      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';

      const sign = dividerSide === 'right' ? 1 : -1;

      const onPointerMove = (ev: PointerEvent) => {
        const delta = ev.clientX - startXRef.current;
        setWidth(Math.max(min, Math.min(max, startWidthRef.current + delta * sign)));
      };

      const onPointerUp = () => {
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        document.removeEventListener('pointermove', onPointerMove);
        document.removeEventListener('pointerup', onPointerUp);

        // Persist final width
        setWidth((w) => {
          try {
            localStorage.setItem(key, String(w));
          } catch {
            // ignore
          }
          return w;
        });
      };

      document.addEventListener('pointermove', onPointerMove);
      document.addEventListener('pointerup', onPointerUp);
    },
    [width, min, max, key, dividerSide],
  );

  return { width, dividerProps: { onPointerDown } };
}
