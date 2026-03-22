import { describe, it, expect } from 'vitest';
import {
  parseMrkdwn,
  isEmojiOnly,
  resolveStandardShortcode,
  type MrkdwnSegment,
} from '@enzyme/shared';

/**
 * Tests that the MrkdwnRenderer's segment handling covers all types
 * produced by parseMrkdwn. Since the renderer is a React Native component
 * that can't be rendered in Node, we verify the parser contract here:
 * every segment type that parseMrkdwn produces must be in the known set
 * that MrkdwnRenderer handles.
 */

const HANDLED_SEGMENT_TYPES = new Set([
  'text',
  'bold',
  'italic',
  'strike',
  'code',
  'code_block',
  'link',
  'user_mention',
  'channel_mention',
  'special_mention',
  'emoji_shortcode',
  'blockquote',
  'bullet_list',
  'ordered_list',
  'line_break',
]);

describe('MrkdwnRenderer segment coverage', () => {
  const cases: [string, string, string[]][] = [
    ['plain text', 'Hello world', ['text']],
    ['bold', '*bold*', ['bold']],
    ['italic', '_italic_', ['italic']],
    ['strikethrough', '~strike~', ['strike']],
    ['inline code', '`code`', ['code']],
    ['code block', '```\ncode\n```', ['code_block']],
    ['link', '<https://example.com|click>', ['link']],
    ['user mention', '<@user1>', ['user_mention']],
    ['channel mention', '<#ch1>', ['channel_mention']],
    ['special mention @here', '<!here>', ['special_mention']],
    ['blockquote', '> quoted', ['blockquote']],
    ['bullet list', '- item 1\n- item 2', ['bullet_list']],
    ['ordered list', '1. first\n2. second', ['ordered_list']],
    ['line break', 'line 1\n\nline 2', ['text', 'line_break']],
  ];

  it.each(cases)('handles %s segment', (_, input, expectedTypes) => {
    const segments = parseMrkdwn(input);
    const types = segments.map((s) => s.type);
    for (const t of expectedTypes) {
      expect(types).toContain(t);
      expect(HANDLED_SEGMENT_TYPES.has(t)).toBe(true);
    }
  });

  it('every segment type from parser is handled by renderer', () => {
    // Parse a comprehensive message that produces many segment types
    const content = [
      '*bold* _italic_ ~strike~ `code`',
      '```\nblock\n```',
      '<https://example.com|link>',
      '<@user1> <#ch1> <!here>',
      '> quoted',
      '- bullet',
      '1. ordered',
      ':smile:',
    ].join('\n');

    const segments = parseMrkdwn(content);
    const allTypes = new Set<string>();
    function collectTypes(segs: MrkdwnSegment[]) {
      for (const s of segs) {
        allTypes.add(s.type);
        if ('segments' in s) collectTypes(s.segments as MrkdwnSegment[]);
        if ('items' in s) for (const item of s.items as MrkdwnSegment[][]) collectTypes(item);
      }
    }
    collectTypes(segments);

    for (const type of allTypes) {
      expect(HANDLED_SEGMENT_TYPES).toContain(type);
    }
  });
});

describe('emoji-only detection for large rendering', () => {
  it('detects single emoji as emoji-only', () => {
    const segments = parseMrkdwn(':smile:');
    expect(isEmojiOnly(segments)).toBe(true);
  });

  it('does not treat text with emoji as emoji-only', () => {
    const segments = parseMrkdwn('hello :smile:');
    expect(isEmojiOnly(segments)).toBe(false);
  });
});

describe('emoji shortcode resolution', () => {
  it('resolves known shortcode to unicode', () => {
    const emoji = resolveStandardShortcode('smile');
    expect(emoji).toBeTruthy();
    expect(typeof emoji).toBe('string');
  });

  it('returns undefined for unknown shortcode', () => {
    const emoji = resolveStandardShortcode('definitely_not_a_real_emoji_shortcode');
    expect(emoji).toBeUndefined();
  });
});
