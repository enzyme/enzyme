import { useState, useEffect, useRef, useCallback } from 'react';
import { View, type StyleProp, type ImageStyle, type ViewStyle } from 'react-native';
import { Image, type ImageContentFit } from 'expo-image';
import { useSignedUrl, getUrl, invalidate } from '@enzyme/shared';

interface AuthImageProps {
  fileId: string;
  style?: StyleProp<ImageStyle>;
  placeholderStyle?: StyleProp<ViewStyle>;
  contentFit?: ImageContentFit;
}

export function AuthImage({
  fileId,
  style,
  placeholderStyle,
  contentFit = 'cover',
}: AuthImageProps) {
  const url = useSignedUrl(fileId);
  const [overrideUrl, setOverrideUrl] = useState<string | null>(null);
  const retryCountRef = useRef(0);
  const mountedRef = useRef(true);
  const [prevFileId, setPrevFileId] = useState(fileId);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // Reset state when fileId changes (synchronous render-phase pattern)
  if (fileId !== prevFileId) {
    setPrevFileId(fileId);
    setOverrideUrl(null);
    retryCountRef.current = 0;
  }

  const handleError = useCallback(() => {
    if (retryCountRef.current >= 1) return;
    retryCountRef.current += 1;
    invalidate(fileId);
    getUrl(fileId)
      .then((newUrl) => {
        if (mountedRef.current) setOverrideUrl(newUrl);
      })
      .catch(() => {});
  }, [fileId]);

  const src = overrideUrl ?? url;

  if (!src) {
    return <View style={[{ backgroundColor: '#e5e7eb' }, placeholderStyle]} />;
  }

  return (
    <Image source={{ uri: src }} style={style} contentFit={contentFit} onError={handleError} />
  );
}
