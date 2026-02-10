import { useState, useEffect, useRef, type ImgHTMLAttributes } from 'react';
import { getCachedIfFresh, getUrl, invalidate } from '../../lib/signedUrlCache';
import { cn } from '../../lib/utils';

interface AuthImageProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'src'> {
  fileId: string;
}

export function AuthImage({ fileId, className, alt, ...props }: AuthImageProps) {
  const [src, setSrc] = useState<string | null>(() => getCachedIfFresh(fileId));
  const [prevFileId, setPrevFileId] = useState(fileId);
  const retryCountRef = useRef(0);
  const mountedRef = useRef(true);

  // Reset state synchronously during render when fileId changes (React-recommended pattern)
  if (fileId !== prevFileId) {
    setPrevFileId(fileId);
    setSrc(getCachedIfFresh(fileId));
  }

  useEffect(() => {
    mountedRef.current = true;
    retryCountRef.current = 0;

    if (getCachedIfFresh(fileId)) return;

    getUrl(fileId)
      .then((url) => {
        if (mountedRef.current) setSrc(url);
      })
      .catch(() => {});

    return () => {
      mountedRef.current = false;
    };
  }, [fileId]);

  function handleError() {
    if (retryCountRef.current >= 1) return; // give up after one retry
    retryCountRef.current += 1;
    invalidate(fileId);
    getUrl(fileId)
      .then((url) => {
        if (mountedRef.current) setSrc(url);
      })
      .catch(() => {});
  }

  if (!src) {
    return (
      <div
        className={cn('animate-pulse bg-gray-200 dark:bg-gray-700', className)}
        role="img"
        aria-label={alt}
      />
    );
  }

  return <img src={src} alt={alt} className={className} onError={handleError} {...props} />;
}
