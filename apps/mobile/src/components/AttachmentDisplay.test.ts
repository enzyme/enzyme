import { describe, it, expect } from 'vitest';
import { isImageType, formatFileSize } from '../lib/attachmentUtils';

describe('isImageType', () => {
  it('returns true for image content types', () => {
    expect(isImageType('image/png')).toBe(true);
    expect(isImageType('image/jpeg')).toBe(true);
    expect(isImageType('image/gif')).toBe(true);
    expect(isImageType('image/webp')).toBe(true);
    expect(isImageType('image/svg+xml')).toBe(true);
  });

  it('returns false for non-image content types', () => {
    expect(isImageType('application/pdf')).toBe(false);
    expect(isImageType('text/plain')).toBe(false);
    expect(isImageType('video/mp4')).toBe(false);
    expect(isImageType('audio/mpeg')).toBe(false);
  });
});

describe('formatFileSize', () => {
  it('formats bytes', () => {
    expect(formatFileSize(0)).toBe('0 B');
    expect(formatFileSize(512)).toBe('512 B');
    expect(formatFileSize(1023)).toBe('1023 B');
  });

  it('formats kilobytes', () => {
    expect(formatFileSize(1024)).toBe('1.0 KB');
    expect(formatFileSize(1536)).toBe('1.5 KB');
    expect(formatFileSize(102400)).toBe('100.0 KB');
  });

  it('formats megabytes', () => {
    expect(formatFileSize(1048576)).toBe('1.0 MB');
    expect(formatFileSize(5242880)).toBe('5.0 MB');
    expect(formatFileSize(1572864)).toBe('1.5 MB');
  });
});

describe('attachment splitting logic', () => {
  function makeAttachment(id: string, contentType: string) {
    return { id, content_type: contentType, filename: `file-${id}`, size_bytes: 1024 };
  }

  it('splits attachments into images and files', () => {
    const attachments = [
      makeAttachment('1', 'image/png'),
      makeAttachment('2', 'application/pdf'),
      makeAttachment('3', 'image/jpeg'),
      makeAttachment('4', 'text/plain'),
    ];

    const images = attachments.filter((a) => isImageType(a.content_type));
    const files = attachments.filter((a) => !isImageType(a.content_type));

    expect(images).toHaveLength(2);
    expect(images[0].id).toBe('1');
    expect(images[1].id).toBe('3');
    expect(files).toHaveLength(2);
    expect(files[0].id).toBe('2');
    expect(files[1].id).toBe('4');
  });

  it('handles all images', () => {
    const attachments = [makeAttachment('1', 'image/png'), makeAttachment('2', 'image/jpeg')];

    const images = attachments.filter((a) => isImageType(a.content_type));
    const files = attachments.filter((a) => !isImageType(a.content_type));

    expect(images).toHaveLength(2);
    expect(files).toHaveLength(0);
  });

  it('handles all files', () => {
    const attachments = [makeAttachment('1', 'application/pdf'), makeAttachment('2', 'text/plain')];

    const images = attachments.filter((a) => isImageType(a.content_type));
    const files = attachments.filter((a) => !isImageType(a.content_type));

    expect(images).toHaveLength(0);
    expect(files).toHaveLength(2);
  });

  it('image grid shows overlay for 5+ images', () => {
    const images = Array.from({ length: 7 }, (_, i) => makeAttachment(String(i), 'image/png'));

    const showOverlay = images.length > 4;
    const visibleCount = showOverlay ? 4 : images.length;
    const remainingCount = images.length - 3;

    expect(showOverlay).toBe(true);
    expect(visibleCount).toBe(4);
    expect(remainingCount).toBe(4); // "+4"
  });

  it('image grid shows no overlay for 4 or fewer images', () => {
    const twoImages = Array.from({ length: 2 }, (_, i) => makeAttachment(String(i), 'image/png'));
    expect(twoImages.length > 4).toBe(false);

    const fourImages = Array.from({ length: 4 }, (_, i) => makeAttachment(String(i), 'image/png'));
    expect(fourImages.length > 4).toBe(false);
  });
});
