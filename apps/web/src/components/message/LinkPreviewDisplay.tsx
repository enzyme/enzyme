import { useState } from 'react';
import type { LinkPreview } from '@enzyme/api-client';
import { DismissPreviewButton } from './DismissPreviewButton';

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
      {onDismiss && <DismissPreviewButton onDismiss={onDismiss} />}
    </div>
  );
}
