import { describe, it, expect } from 'vitest';
import { parseMrkdwn, type MrkdwnSegment } from './parser';

describe('parseMrkdwn', () => {
  describe('basic text', () => {
    it('returns empty array for empty string', () => {
      expect(parseMrkdwn('')).toEqual([]);
    });

    it('returns text segment for plain text', () => {
      expect(parseMrkdwn('Hello world')).toEqual([{ type: 'text', content: 'Hello world' }]);
    });

    it('handles multiple lines', () => {
      const result = parseMrkdwn('Line 1\nLine 2');
      expect(result).toContainEqual({ type: 'text', content: 'Line 1' });
      expect(result).toContainEqual({ type: 'line_break' });
      expect(result).toContainEqual({ type: 'text', content: 'Line 2' });
    });
  });

  describe('inline formatting', () => {
    it('parses bold text', () => {
      expect(parseMrkdwn('*bold*')).toEqual([{ type: 'bold', content: 'bold' }]);
    });

    it('parses italic text', () => {
      expect(parseMrkdwn('_italic_')).toEqual([{ type: 'italic', content: 'italic' }]);
    });

    it('parses strikethrough text', () => {
      expect(parseMrkdwn('~strikethrough~')).toEqual([
        { type: 'strike', content: 'strikethrough' },
      ]);
    });

    it('parses inline code', () => {
      expect(parseMrkdwn('`code`')).toEqual([{ type: 'code', content: 'code' }]);
    });

    it('parses mixed inline formatting', () => {
      const result = parseMrkdwn('Hello *bold* and _italic_');
      expect(result).toEqual([
        { type: 'text', content: 'Hello ' },
        { type: 'bold', content: 'bold' },
        { type: 'text', content: ' and ' },
        { type: 'italic', content: 'italic' },
      ]);
    });

    it('handles unclosed formatting markers as text', () => {
      const result = parseMrkdwn('*unclosed');
      expect(result).toEqual([{ type: 'text', content: '*unclosed' }]);
    });
  });

  describe('code blocks', () => {
    it('parses code block without language', () => {
      const result = parseMrkdwn('```\nconst x = 1;\n```');
      expect(result).toEqual([
        { type: 'code_block', content: 'const x = 1;', language: undefined },
      ]);
    });

    it('parses code block with language', () => {
      const result = parseMrkdwn('```javascript\nconst x = 1;\n```');
      expect(result).toEqual([
        { type: 'code_block', content: 'const x = 1;', language: 'javascript' },
      ]);
    });

    it('preserves multiple lines in code block', () => {
      const result = parseMrkdwn('```\nline 1\nline 2\nline 3\n```');
      expect(result).toEqual([
        { type: 'code_block', content: 'line 1\nline 2\nline 3', language: undefined },
      ]);
    });
  });

  describe('blockquotes', () => {
    it('parses single line blockquote', () => {
      const result = parseMrkdwn('> quoted text');
      expect(result).toEqual([
        {
          type: 'blockquote',
          segments: [{ type: 'text', content: 'quoted text' }],
        },
      ]);
    });

    it('parses multi-line blockquote', () => {
      const result = parseMrkdwn('> line 1\n> line 2');
      expect(result[0].type).toBe('blockquote');
      const blockquote = result[0] as { type: 'blockquote'; segments: MrkdwnSegment[] };
      // Parser combines multi-line blockquote content into single text segment
      expect(blockquote.segments).toContainEqual({ type: 'text', content: 'line 1\nline 2' });
    });

    it('parses blockquote with inline formatting', () => {
      const result = parseMrkdwn('> *bold* text');
      expect(result[0].type).toBe('blockquote');
      const blockquote = result[0] as { type: 'blockquote'; segments: MrkdwnSegment[] };
      expect(blockquote.segments).toContainEqual({ type: 'bold', content: 'bold' });
    });
  });

  describe('lists', () => {
    it('parses bullet list with bullet character', () => {
      const result = parseMrkdwn('• Item 1\n• Item 2');
      expect(result).toEqual([
        {
          type: 'bullet_list',
          items: [[{ type: 'text', content: 'Item 1' }], [{ type: 'text', content: 'Item 2' }]],
        },
      ]);
    });

    it('parses bullet list with dash', () => {
      const result = parseMrkdwn('- Item 1\n- Item 2');
      expect(result).toEqual([
        {
          type: 'bullet_list',
          items: [[{ type: 'text', content: 'Item 1' }], [{ type: 'text', content: 'Item 2' }]],
        },
      ]);
    });

    it('parses ordered list', () => {
      const result = parseMrkdwn('1. First\n2. Second\n3. Third');
      expect(result).toEqual([
        {
          type: 'ordered_list',
          items: [
            [{ type: 'text', content: 'First' }],
            [{ type: 'text', content: 'Second' }],
            [{ type: 'text', content: 'Third' }],
          ],
        },
      ]);
    });

    it('parses list items with inline formatting', () => {
      const result = parseMrkdwn('- *bold* item');
      expect(result[0].type).toBe('bullet_list');
      const list = result[0] as { type: 'bullet_list'; items: MrkdwnSegment[][] };
      expect(list.items[0]).toContainEqual({ type: 'bold', content: 'bold' });
    });
  });

  describe('mentions', () => {
    it('parses user mention', () => {
      expect(parseMrkdwn('<@user123>')).toEqual([{ type: 'user_mention', userId: 'user123' }]);
    });

    it('parses special mention @here', () => {
      expect(parseMrkdwn('<!here>')).toEqual([{ type: 'special_mention', mentionType: 'here' }]);
    });

    it('parses special mention @channel', () => {
      expect(parseMrkdwn('<!channel>')).toEqual([
        { type: 'special_mention', mentionType: 'channel' },
      ]);
    });

    it('parses special mention @everyone', () => {
      expect(parseMrkdwn('<!everyone>')).toEqual([
        { type: 'special_mention', mentionType: 'everyone' },
      ]);
    });

    it('parses channel mention', () => {
      expect(parseMrkdwn('<#channel123>')).toEqual([
        { type: 'channel_mention', channelId: 'channel123' },
      ]);
    });

    it('parses mention in text', () => {
      const result = parseMrkdwn('Hello <@user1> and <@user2>!');
      expect(result).toEqual([
        { type: 'text', content: 'Hello ' },
        { type: 'user_mention', userId: 'user1' },
        { type: 'text', content: ' and ' },
        { type: 'user_mention', userId: 'user2' },
        { type: 'text', content: '!' },
      ]);
    });
  });

  describe('links', () => {
    it('parses link with text', () => {
      expect(parseMrkdwn('<https://example.com|Example>')).toEqual([
        { type: 'link', url: 'https://example.com', text: 'Example' },
      ]);
    });

    it('parses link without text', () => {
      expect(parseMrkdwn('<https://example.com>')).toEqual([
        { type: 'link', url: 'https://example.com', text: 'https://example.com' },
      ]);
    });

    it('parses http link', () => {
      expect(parseMrkdwn('<http://example.com>')).toEqual([
        { type: 'link', url: 'http://example.com', text: 'http://example.com' },
      ]);
    });

    it('parses link in text', () => {
      const result = parseMrkdwn('Check out <https://example.com|this link>!');
      expect(result).toEqual([
        { type: 'text', content: 'Check out ' },
        { type: 'link', url: 'https://example.com', text: 'this link' },
        { type: 'text', content: '!' },
      ]);
    });
  });

  describe('complex content', () => {
    it('parses message with mentions and formatting', () => {
      const result = parseMrkdwn(
        'Hey <@user1>, check out this *important* link: <https://example.com|click here>',
      );
      expect(result).toContainEqual({ type: 'user_mention', userId: 'user1' });
      expect(result).toContainEqual({ type: 'bold', content: 'important' });
      expect(result).toContainEqual({
        type: 'link',
        url: 'https://example.com',
        text: 'click here',
      });
    });

    it('handles text before code block', () => {
      const result = parseMrkdwn('Here is some code:\n```\ncode\n```');
      expect(result[0]).toEqual({ type: 'text', content: 'Here is some code:' });
      expect(result).toContainEqual({ type: 'code_block', content: 'code', language: undefined });
    });

    it('handles empty lines as line breaks', () => {
      const result = parseMrkdwn('Paragraph 1\n\nParagraph 2');
      expect(result).toContainEqual({ type: 'line_break' });
    });
  });
});
