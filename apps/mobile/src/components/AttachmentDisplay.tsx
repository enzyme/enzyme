import { useState } from 'react';
import { View, Text, Pressable, ActivityIndicator } from 'react-native';
import { cacheDirectory, downloadAsync } from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';
import type { Attachment } from '@enzyme/api-client';
import { useSignedUrl } from '@enzyme/shared';
import { AuthImage } from './AuthImage';

interface AttachmentDisplayProps {
  attachments: Attachment[];
  onImagePress?: (images: Attachment[], index: number) => void;
}

function isImageType(contentType: string): boolean {
  return contentType.startsWith('image/');
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// --- FileAttachment ---

function FileAttachment({ attachment }: { attachment: Attachment }) {
  const url = useSignedUrl(attachment.id);
  const [downloading, setDownloading] = useState(false);

  async function handlePress() {
    if (!url || downloading) return;
    setDownloading(true);
    try {
      const localUri = cacheDirectory + attachment.filename;
      const { uri } = await downloadAsync(url, localUri);
      await Sharing.shareAsync(uri);
    } catch {
      // Download or share failed silently
    } finally {
      setDownloading(false);
    }
  }

  return (
    <Pressable
      className="flex-row items-center rounded-lg border border-neutral-200 px-3 py-2 active:bg-neutral-50 dark:border-neutral-700 dark:active:bg-neutral-800"
      onPress={handlePress}
      disabled={!url}
      style={{ opacity: url ? 1 : 0.5 }}
    >
      <View className="mr-3 h-10 w-10 items-center justify-center rounded-lg bg-neutral-100 dark:bg-neutral-700">
        {downloading ? (
          <ActivityIndicator size="small" />
        ) : (
          <Text className="text-lg text-neutral-500 dark:text-neutral-400">📄</Text>
        )}
      </View>
      <View className="min-w-0 flex-1">
        <Text className="text-sm font-medium text-neutral-900 dark:text-white" numberOfLines={1}>
          {attachment.filename}
        </Text>
        <Text className="text-xs text-neutral-500 dark:text-neutral-400">
          {formatFileSize(attachment.size_bytes)}
        </Text>
      </View>
    </Pressable>
  );
}

// --- ImageThumbnail ---

function ImageThumbnail({
  attachment,
  onPress,
  aspectRatio,
}: {
  attachment: Attachment;
  onPress: () => void;
  aspectRatio?: number;
}) {
  return (
    <Pressable
      className="overflow-hidden rounded-lg"
      onPress={onPress}
      style={aspectRatio ? { aspectRatio } : undefined}
    >
      <AuthImage
        fileId={attachment.id}
        style={{ width: '100%', height: '100%' }}
        placeholderStyle={
          aspectRatio ? { width: '100%', aspectRatio } : { width: '100%', height: 200 }
        }
        contentFit="cover"
      />
    </Pressable>
  );
}

// --- ImageGrid ---

function ImageGrid({
  images,
  onImagePress,
}: {
  images: Attachment[];
  onImagePress?: (images: Attachment[], index: number) => void;
}) {
  const showOverlay = images.length > 4;
  const visibleCount = showOverlay ? 4 : images.length;
  const visibleImages = images.slice(0, visibleCount);
  const remainingCount = images.length - 3;

  return (
    <View className="flex-row flex-wrap" style={{ gap: 4 }}>
      {visibleImages.map((image, index) => {
        const isOverlayCell = showOverlay && index === 3;

        return (
          <View key={image.id} style={{ width: '48.5%', aspectRatio: 1 }}>
            <Pressable
              className="flex-1 overflow-hidden rounded-lg"
              onPress={() => onImagePress?.(images, isOverlayCell ? 3 : index)}
            >
              <AuthImage
                fileId={image.id}
                style={{ width: '100%', height: '100%' }}
                placeholderStyle={{ width: '100%', height: '100%' }}
                contentFit="cover"
              />
              {isOverlayCell && (
                <View className="absolute inset-0 items-center justify-center bg-black/50">
                  <Text className="text-2xl font-semibold text-white">+{remainingCount}</Text>
                </View>
              )}
            </Pressable>
          </View>
        );
      })}
    </View>
  );
}

// --- AttachmentDisplay (exported) ---

export function AttachmentDisplay({ attachments, onImagePress }: AttachmentDisplayProps) {
  if (!attachments || attachments.length === 0) return null;

  const images = attachments.filter((a) => isImageType(a.content_type));
  const files = attachments.filter((a) => !isImageType(a.content_type));

  return (
    <View className="mt-2" style={{ gap: 8 }}>
      {images.length === 1 && (
        <ImageThumbnail attachment={images[0]} onPress={() => onImagePress?.(images, 0)} />
      )}
      {images.length > 1 && <ImageGrid images={images} onImagePress={onImagePress} />}
      {files.length > 0 && (
        <View style={{ gap: 8 }}>
          {files.map((attachment) => (
            <FileAttachment key={attachment.id} attachment={attachment} />
          ))}
        </View>
      )}
    </View>
  );
}
