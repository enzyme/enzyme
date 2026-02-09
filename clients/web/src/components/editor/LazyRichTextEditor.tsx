import { lazy, Suspense, forwardRef } from 'react';
import type { RichTextEditorRef, RichTextEditorProps } from './RichTextEditor';

const RichTextEditorLazy = lazy(() =>
  import('./RichTextEditor').then((m) => ({ default: m.RichTextEditor }))
);

function EditorSkeleton() {
  return (
    <div className="border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-900">
      {/* Toolbar placeholder — matches real toolbar: gap-0.5 px-2 py-1, buttons are p-1.5 + w-4 h-4 icon */}
      <div className="flex items-center gap-0.5 px-2 py-1 bg-gray-50 dark:bg-gray-800 rounded-t-lg">
        <div className="h-7 w-7 rounded" />
        <div className="h-7 w-7 rounded" />
        <div className="h-7 w-7 rounded" />
        <div className="h-7 w-7 rounded" />
      </div>
      {/* Content area placeholder */}
      <div className="px-4 py-3 min-h-[2.5rem]">
        <div className="h-6 w-32 rounded bg-gray-100 dark:bg-gray-800" />
      </div>
      {/* Action row placeholder — matches real: justify-between px-2 py-1 */}
      <div className="flex items-center justify-between px-2 py-1">
        <div className="flex items-center gap-0.5">
          <div className="h-7 w-7 rounded" />
          <div className="h-7 w-7 rounded" />
          <div className="h-7 w-7 rounded" />
          <div className="h-7 w-7 rounded" />
        </div>
        <div className="h-7 w-7 rounded" />
      </div>
    </div>
  );
}

export const LazyRichTextEditor = forwardRef<RichTextEditorRef, RichTextEditorProps>(
  (props, ref) => (
    <Suspense fallback={<EditorSkeleton />}>
      <RichTextEditorLazy ref={ref} {...props} />
    </Suspense>
  )
);

LazyRichTextEditor.displayName = 'LazyRichTextEditor';
