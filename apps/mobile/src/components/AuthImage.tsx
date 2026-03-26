import { useState, useEffect, useRef } from 'react';
import { View, type StyleProp, type ImageStyle, type ViewStyle } from 'react-native';
import { Image, type ImageContentFit } from 'expo-image';
import { useSignedUrl, getUrl, invalidate } from '@enzyme/shared';

interface AuthImageProps {
  fileId: string;
  style?: StyleProp<ImageStyle>;
  placeholderStyle?: StyleProp<ViewStyle>;
  contentFit?: ImageContentFit;
  className?: string;
}

export function AuthImage({
  fileId,
  style,
  placeholderStyle,
  contentFit = 'cover',
  className,
}: AuthImageProps) {
  const url = useSignedUrl(fileId);
  const [src, setSrc] = useState<string | null>(url);
  const retryCountRef = useRef(0);
  const prevFileIdRef = useRef(fileId);

  // Sync src with the hook's url
  useEffect(() => {
    setSrc(url);
  }, [url]);

  // Reset retry count when fileId changes
  if (fileId !== prevFileIdRef.current) {
    prevFileIdRef.current = fileId;
    retryCountRef.current = 0;
  }

  function handleError() {
    if (retryCountRef.current >= 1) return;
    retryCountRef.current += 1;
    invalidate(fileId);
    getUrl(fileId)
      .then((newUrl) => setSrc(newUrl))
      .catch(() => {});
  }

  if (!src) {
    return (
      <View className={className} style={[{ backgroundColor: '#e5e7eb' }, placeholderStyle]} />
    );
  }

  return (
    <Image
      source={{ uri: src }}
      style={style}
      className={className}
      contentFit={contentFit}
      onError={handleError}
    />
  );
}
