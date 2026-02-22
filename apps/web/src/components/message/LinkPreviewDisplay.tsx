import { useState } from 'react';
import type { LinkPreview } from '@enzyme/api-client';

interface LinkPreviewDisplayProps {
  preview: LinkPreview;
  onDismiss?: () => void;
}

export function LinkPreviewDisplay({ preview, onDismiss }: LinkPreviewDisplayProps) {
  const [imageError, setImageError] = useState(false);

  if (!preview.title && !preview.description) return null;

  const showImage = preview.image_url && !imageError;

  return (
    <div className="group/preview relative mt-2 max-w-lg">
      <a
        href={preview.url}
        target="_blank"
        rel="noopener noreferrer"
        className="block overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
      >
        {showImage && (
          <img
            src={preview.image_url}
            alt=""
            className="max-h-52 w-full object-cover"
            onError={() => setImageError(true)}
          />
        )}
        <div className="px-3 py-2">
          {preview.site_name && (
            <p className="truncate text-xs font-medium text-blue-600 dark:text-blue-400">
              {preview.site_name}
            </p>
          )}
          {preview.title && (
            <p className="line-clamp-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
              {preview.title}
            </p>
          )}
          {preview.description && (
            <p className="mt-0.5 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
              {preview.description}
            </p>
          )}
        </div>
      </a>
      {onDismiss && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDismiss();
          }}
          className="absolute -top-2 -right-2 hidden cursor-pointer rounded-full border border-gray-200 bg-white p-0.5 text-gray-400 shadow-sm group-hover/preview:block hover:bg-gray-100 hover:text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-500 dark:hover:bg-gray-800 dark:hover:text-gray-300"
          aria-label="Remove link preview"
        >
          <svg
            className="h-3.5 w-3.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}
