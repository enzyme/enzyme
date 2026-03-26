import { useState } from 'react';
import { View, Text, Pressable, Linking } from 'react-native';
import { Image } from 'expo-image';
import type { LinkPreview as LinkPreviewType } from '@enzyme/api-client';

interface LinkPreviewProps {
  preview: LinkPreviewType;
}

export function LinkPreview({ preview }: LinkPreviewProps) {
  const [imageError, setImageError] = useState(false);

  if (!preview.title && !preview.description) return null;

  const showImage = preview.image_url && !imageError;

  function handlePress() {
    if (preview.url) {
      Linking.openURL(preview.url);
    }
  }

  return (
    <Pressable
      className="mt-2 overflow-hidden rounded-lg border border-neutral-200 bg-white active:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:active:bg-neutral-700"
      onPress={handlePress}
    >
      {showImage && (
        <Image
          source={{ uri: preview.image_url }}
          style={{ width: '100%', height: 160 }}
          contentFit="cover"
          onError={() => setImageError(true)}
        />
      )}
      <View className="px-3 py-2">
        {preview.site_name && (
          <Text className="text-xs font-medium text-blue-600 dark:text-blue-400" numberOfLines={1}>
            {preview.site_name}
          </Text>
        )}
        {preview.title && (
          <Text
            className="text-sm font-semibold text-neutral-900 dark:text-neutral-100"
            numberOfLines={2}
          >
            {preview.title}
          </Text>
        )}
        {preview.description && (
          <Text className="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400" numberOfLines={2}>
            {preview.description}
          </Text>
        )}
      </View>
    </Pressable>
  );
}
