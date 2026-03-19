import type { MrkdwnSegment } from './parser';

export function isEmojiOnly(segments: MrkdwnSegment[]): boolean {
  const meaningful = segments.filter(
    (s) =>
      !(s.type === 'text' && 'content' in s && s.content.trim() === '') && s.type !== 'line_break',
  );
  if (meaningful.length === 0) return false;
  if (meaningful.some((s) => s.type !== 'emoji_shortcode')) return false;
  return meaningful.length >= 1 && meaningful.length <= 3;
}
