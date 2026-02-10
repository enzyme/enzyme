import { useState, useEffect } from 'react';
import {
  Button as AriaButton,
  Dialog,
  Heading,
  Modal as AriaModal,
  ModalOverlay,
} from 'react-aria-components';
import type { Attachment } from '@feather/api-client';
import { AuthImage } from '../ui';
import { useSignedUrl } from '../../hooks/useSignedUrl';
import { cn } from '../../lib/utils';

interface AttachmentDisplayProps {
  attachments: Attachment[];
}

function isImageType(contentType: string): boolean {
  return contentType.startsWith('image/');
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// --- CarouselDownloadLink ---
// Extracted as a component so the useSignedUrl hook re-runs when `fileId` changes.

function CarouselDownloadLink({ fileId, filename }: { fileId: string; filename: string }) {
  const url = useSignedUrl(fileId);

  return (
    <a
      href={url ?? '#'}
      download={filename}
      className={cn(
        'rounded-lg bg-black/50 p-2 text-white transition-colors hover:bg-black/70',
        !url && 'pointer-events-none opacity-50',
      )}
      title="Download"
      onClick={(e) => e.stopPropagation()}
    >
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
        />
      </svg>
    </a>
  );
}

// --- ImageCarousel ---

interface ImageCarouselProps {
  images: Attachment[];
  initialIndex: number;
  isOpen: boolean;
  onClose: () => void;
}

function ImageCarousel({ images, initialIndex, isOpen, onClose }: ImageCarouselProps) {
  const [currentIndex, setCurrentIndex] = useState(initialIndex);

  useEffect(() => {
    if (!isOpen) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        setCurrentIndex((i) => (i === 0 ? images.length - 1 : i - 1));
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        setCurrentIndex((i) => (i === images.length - 1 ? 0 : i + 1));
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, images.length]);

  const current = images[currentIndex];
  if (!current) return null;

  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
      isDismissable
    >
      <AriaModal className="max-h-[90vh] max-w-[90vw] outline-none">
        <Dialog className="outline-none" aria-label="Image viewer">
          <div className="relative flex items-center justify-center">
            {images.length > 1 && (
              <AriaButton
                onPress={() => setCurrentIndex((i) => (i === 0 ? images.length - 1 : i - 1))}
                className="absolute left-2 z-10 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70"
                aria-label="Previous image"
              >
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 19l-7-7 7-7"
                  />
                </svg>
              </AriaButton>
            )}

            <AuthImage
              fileId={current.id}
              alt={current.filename}
              className="max-h-[85vh] max-w-full rounded-lg object-contain"
            />

            {images.length > 1 && (
              <AriaButton
                onPress={() => setCurrentIndex((i) => (i === images.length - 1 ? 0 : i + 1))}
                className="absolute right-2 z-10 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70"
                aria-label="Next image"
              >
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </AriaButton>
            )}

            <div className="absolute right-2 top-2 flex gap-2">
              <CarouselDownloadLink fileId={current.id} filename={current.filename} />
              <button
                onClick={onClose}
                className="rounded-lg bg-black/50 p-2 text-white transition-colors hover:bg-black/70"
                title="Close"
              >
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            <div className="absolute bottom-2 left-2 right-2 text-center">
              <Heading
                slot="title"
                className="inline-block rounded-lg bg-black/50 px-3 py-1 text-sm text-white"
              >
                {current.filename}
              </Heading>
              {images.length > 1 && (
                <p className="mt-1 text-xs text-white" data-testid="carousel-counter">
                  {currentIndex + 1} of {images.length}
                </p>
              )}
            </div>
          </div>
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}

// --- ImageGrid ---

interface ImageGridProps {
  images: Attachment[];
}

function ImageGrid({ images }: ImageGridProps) {
  const [isCarouselOpen, setIsCarouselOpen] = useState(false);
  const [carouselStartIndex, setCarouselStartIndex] = useState(0);
  const [carouselKey, setCarouselKey] = useState(0);

  function openCarousel(index: number) {
    setCarouselStartIndex(index);
    setCarouselKey((k) => k + 1);
    setIsCarouselOpen(true);
  }

  // Show at most 4 cells; 4th is overlay if 5+ images
  const showOverlay = images.length > 4;
  const visibleCount = showOverlay ? 4 : images.length;
  const visibleImages = images.slice(0, visibleCount);
  const remainingCount = images.length - 3; // for the "+N" overlay

  return (
    <>
      <div
        className="grid max-w-md grid-cols-2 gap-1 overflow-hidden rounded-lg"
        data-testid="image-grid"
      >
        {visibleImages.map((image, index) => {
          const isOverlayCell = showOverlay && index === 3;

          return (
            <button
              key={image.id}
              onClick={() => openCarousel(isOverlayCell ? 3 : index)}
              className={cn(
                'relative cursor-pointer overflow-hidden focus:outline-none focus:ring-2 focus:ring-blue-500',
                // Layout: 3 images → first spans full width
                images.length === 3 && index === 0 && 'col-span-2 aspect-video',
                // All other cells are square
                !(images.length === 3 && index === 0) && 'aspect-square',
              )}
              aria-label={
                isOverlayCell ? `View ${remainingCount} more images` : `View ${image.filename}`
              }
            >
              <AuthImage
                fileId={image.id}
                alt={image.filename}
                className="h-full w-full object-cover"
                loading="lazy"
              />
              {isOverlayCell && (
                <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                  <span className="text-2xl font-semibold text-white" data-testid="overflow-count">
                    +{remainingCount}
                  </span>
                </div>
              )}
            </button>
          );
        })}
      </div>
      <ImageCarousel
        key={carouselKey}
        images={images}
        initialIndex={carouselStartIndex}
        isOpen={isCarouselOpen}
        onClose={() => setIsCarouselOpen(false)}
      />
    </>
  );
}

// --- ImageAttachment (single image thumbnail) ---

function ImageAttachment({ attachment, onClick }: { attachment: Attachment; onClick: () => void }) {
  return (
    <AriaButton
      onPress={onClick}
      className="block cursor-pointer overflow-hidden rounded-lg border border-gray-200 transition-colors hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600"
      aria-label={`View ${attachment.filename}`}
    >
      <AuthImage
        fileId={attachment.id}
        alt={attachment.filename}
        className="max-h-64 max-w-full object-contain"
        loading="lazy"
      />
    </AriaButton>
  );
}

// --- FileAttachment ---

function FileAttachment({ attachment }: { attachment: Attachment }) {
  const url = useSignedUrl(attachment.id);

  return (
    <a
      href={url ?? '#'}
      download={attachment.filename}
      className={cn(
        'flex max-w-xs items-center gap-3 rounded-lg border border-gray-200 px-3 py-2 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800',
        !url && 'pointer-events-none opacity-50',
      )}
    >
      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700">
        <svg
          className="h-5 w-5 text-gray-500 dark:text-gray-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
          />
        </svg>
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-gray-900 dark:text-white">
          {attachment.filename}
        </p>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          {formatFileSize(attachment.size_bytes)}
        </p>
      </div>
      <svg
        className="h-4 w-4 flex-shrink-0 text-gray-400"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
        />
      </svg>
    </a>
  );
}

// --- SingleImageSection ---

function SingleImageSection({ image }: { image: Attachment }) {
  const [isCarouselOpen, setIsCarouselOpen] = useState(false);

  return (
    <>
      <ImageAttachment attachment={image} onClick={() => setIsCarouselOpen(true)} />
      <ImageCarousel
        images={[image]}
        initialIndex={0}
        isOpen={isCarouselOpen}
        onClose={() => setIsCarouselOpen(false)}
      />
    </>
  );
}

// --- AttachmentDisplay (exported) ---

export function AttachmentDisplay({ attachments }: AttachmentDisplayProps) {
  if (!attachments || attachments.length === 0) return null;

  const images = attachments.filter((a) => isImageType(a.content_type));
  const files = attachments.filter((a) => !isImageType(a.content_type));

  return (
    <div className="mt-2 space-y-2">
      {images.length === 1 && <SingleImageSection image={images[0]} />}
      {images.length > 1 && <ImageGrid images={images} />}
      {files.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {files.map((attachment) => (
            <FileAttachment key={attachment.id} attachment={attachment} />
          ))}
        </div>
      )}
    </div>
  );
}
