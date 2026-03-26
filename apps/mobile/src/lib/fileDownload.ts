import { cacheDirectory, downloadAsync } from 'expo-file-system/legacy';
import { getUrl } from '@enzyme/shared';

/** Sanitize a filename by stripping path separators and null bytes. */
function safeFilename(filename: string): string {
  const base = filename.split(/[/\\]/).pop() || 'download';
  return base.replace(/\0/g, '') || 'download';
}

/** Download a file to the cache directory and return the local URI. Uses file ID in the path to prevent collisions. */
export async function downloadToCache(fileId: string, filename: string): Promise<string> {
  const url = await getUrl(fileId);
  const localUri = cacheDirectory + fileId + '_' + safeFilename(filename);
  const { uri } = await downloadAsync(url, localUri);
  return uri;
}
