import { useState, useRef, useCallback } from 'react';
import {
  View,
  Text,
  Pressable,
  Modal,
  FlatList,
  useWindowDimensions,
  type ListRenderItemInfo,
  type NativeSyntheticEvent,
  type NativeScrollEvent,
  Alert,
} from 'react-native';
import { StatusBar } from 'expo-status-bar';
import * as Sharing from 'expo-sharing';
import * as MediaLibrary from 'expo-media-library';
import type { Attachment } from '@enzyme/api-client';
import { downloadToCache } from '../lib/fileDownload';
import { AuthImage } from './AuthImage';

interface ImageViewerProps {
  images: Attachment[];
  initialIndex: number;
  onClose: () => void;
}

function ActionButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable className="rounded-full bg-black/50 px-4 py-2 active:bg-black/70" onPress={onPress}>
      <Text className="text-sm font-medium text-white">{label}</Text>
    </Pressable>
  );
}

const keyExtractor = (item: Attachment) => item.id;

export function ImageViewer({ images, initialIndex, onClose }: ImageViewerProps) {
  const { width, height } = useWindowDimensions();
  const [currentIndex, setCurrentIndex] = useState(initialIndex);
  const flatListRef = useRef<FlatList>(null);

  const handleScroll = useCallback(
    (e: NativeSyntheticEvent<NativeScrollEvent>) => {
      const offset = e.nativeEvent.contentOffset.x;
      const index = Math.round(offset / width);
      if (index >= 0 && index < images.length) {
        setCurrentIndex(index);
      }
    },
    [width, images.length],
  );

  const currentImage = images[currentIndex];

  async function handleShare() {
    if (!currentImage) return;
    try {
      const uri = await downloadToCache(currentImage.id, currentImage.filename);
      await Sharing.shareAsync(uri);
    } catch {
      Alert.alert('Error', 'Failed to share the image.');
    }
  }

  async function handleSave() {
    if (!currentImage) return;
    try {
      const { status } = await MediaLibrary.requestPermissionsAsync();
      if (status !== 'granted') {
        Alert.alert(
          'Permission needed',
          'Please allow access to save images to your photo library.',
        );
        return;
      }
      const uri = await downloadToCache(currentImage.id, currentImage.filename);
      await MediaLibrary.saveToLibraryAsync(uri);
      Alert.alert('Saved', 'Image saved to your photo library.');
    } catch {
      Alert.alert('Error', 'Failed to save the image.');
    }
  }

  const renderItem = useCallback(
    ({ item }: ListRenderItemInfo<Attachment>) => (
      <View style={{ width, height, justifyContent: 'center', alignItems: 'center' }}>
        <AuthImage fileId={item.id} style={{ width, height }} contentFit="contain" />
      </View>
    ),
    [width, height],
  );

  return (
    <Modal visible transparent animationType="fade" onRequestClose={onClose} statusBarTranslucent>
      <StatusBar hidden />
      <View className="flex-1 bg-black">
        {/* Header overlay */}
        <View className="absolute left-0 right-0 top-0 z-10 flex-row items-center justify-between px-4 pb-3 pt-14">
          <Pressable className="rounded-full bg-black/50 p-2 active:bg-black/70" onPress={onClose}>
            <Text className="text-lg font-bold text-white">✕</Text>
          </Pressable>
          {images.length > 1 && (
            <Text className="rounded-full bg-black/50 px-3 py-1 text-sm text-white">
              {currentIndex + 1} of {images.length}
            </Text>
          )}
          <View style={{ width: 36 }} />
        </View>

        {/* Image pages */}
        <FlatList
          ref={flatListRef}
          data={images}
          renderItem={renderItem}
          keyExtractor={keyExtractor}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          onMomentumScrollEnd={handleScroll}
          initialScrollIndex={initialIndex}
          getItemLayout={(_, index) => ({ length: width, offset: width * index, index })}
          windowSize={3}
          initialNumToRender={1}
          maxToRenderPerBatch={2}
        />

        {/* Footer overlay */}
        <View
          className="absolute bottom-0 left-0 right-0 z-10 flex-row items-center justify-center pb-12 pt-3"
          style={{ gap: 16 }}
        >
          <ActionButton label="Share" onPress={handleShare} />
          <ActionButton label="Save" onPress={handleSave} />
        </View>
      </View>
    </Modal>
  );
}
