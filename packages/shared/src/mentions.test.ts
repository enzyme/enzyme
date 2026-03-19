import { describe, it, expect } from 'vitest';
import {
  parseMentionTrigger,
  insertMention,
  convertMentionsForStorage,
  parseStoredMentions,
  type MentionOption,
  type MentionTrigger,
} from './mentions';

describe('parseMentionTrigger', () => {
  it('detects @ at start of string', () => {
    const result = parseMentionTrigger('@john', 5);
    expect(result).toEqual({
      isActive: true,
      query: 'john',
      startIndex: 0,
    });
  });

  it('detects @ after space', () => {
    const result = parseMentionTrigger('hello @john', 11);
    expect(result).toEqual({
      isActive: true,
      query: 'john',
      startIndex: 6,
    });
  });

  it('detects @ after newline', () => {
    const result = parseMentionTrigger('hello\n@john', 11);
    expect(result).toEqual({
      isActive: true,
      query: 'john',
      startIndex: 6,
    });
  });

  it('returns query text after @', () => {
    const result = parseMentionTrigger('@jo', 3);
    expect(result?.query).toBe('jo');
  });

  it('returns null when no active trigger', () => {
    const result = parseMentionTrigger('hello world', 11);
    expect(result).toBeNull();
  });

  it('returns null when @ is mid-word', () => {
    const result = parseMentionTrigger('test@example.com', 16);
    expect(result).toBeNull();
  });

  it('returns null when cursor is after space following @mention', () => {
    // Cursor at position 6 is after the space
    const result = parseMentionTrigger('@john ', 6);
    expect(result).toBeNull();
  });
});

describe('insertMention', () => {
  it('replaces trigger with @DisplayName ', () => {
    const trigger: MentionTrigger = {
      isActive: true,
      query: 'jo',
      startIndex: 0,
    };
    const mention: MentionOption = {
      type: 'user',
      id: 'user123',
      displayName: 'John Doe',
    };

    const result = insertMention('@jo', trigger, mention);
    expect(result.content).toBe('@John Doe ');
  });

  it('returns correct new cursor position', () => {
    const trigger: MentionTrigger = {
      isActive: true,
      query: 'jo',
      startIndex: 6,
    };
    const mention: MentionOption = {
      type: 'user',
      id: 'user123',
      displayName: 'John Doe',
    };

    const result = insertMention('hello @jo', trigger, mention);
    // "hello " (6) + "@John Doe " (10) = 16
    expect(result.cursorPos).toBe(16);
  });

  it('preserves text before and after', () => {
    const trigger: MentionTrigger = {
      isActive: true,
      query: 'jo',
      startIndex: 6,
    };
    const mention: MentionOption = {
      type: 'user',
      id: 'user123',
      displayName: 'John',
    };

    const result = insertMention('hello @jo world', trigger, mention);
    expect(result.content).toBe('hello @John  world');
  });
});

describe('convertMentionsForStorage', () => {
  it('converts @DisplayName to <@userId>', () => {
    const mentionMap = new Map([['John Doe', 'user123']]);
    // Mentions must be followed by space or end of string
    const result = convertMentionsForStorage('Hello @John Doe there', mentionMap);
    expect(result).toBe('Hello <@user123> there');
  });

  it('converts @here to <!here>', () => {
    const mentionMap = new Map<string, string>();
    const result = convertMentionsForStorage('Hey @here', mentionMap);
    expect(result).toBe('Hey <!here>');
  });

  it('converts @channel to <!channel>', () => {
    const mentionMap = new Map<string, string>();
    const result = convertMentionsForStorage('Attention @channel', mentionMap);
    expect(result).toBe('Attention <!channel>');
  });

  it('converts @everyone to <!everyone>', () => {
    const mentionMap = new Map<string, string>();
    const result = convertMentionsForStorage('@everyone please read', mentionMap);
    expect(result).toBe('<!everyone> please read');
  });

  it('handles multiple mentions in one message', () => {
    const mentionMap = new Map([
      ['John', 'user1'],
      ['Jane', 'user2'],
    ]);
    const result = convertMentionsForStorage('@John and @Jane check this', mentionMap);
    expect(result).toBe('<@user1> and <@user2> check this');
  });

  it('preserves non-mention text', () => {
    const mentionMap = new Map<string, string>();
    const result = convertMentionsForStorage('Just regular text here', mentionMap);
    expect(result).toBe('Just regular text here');
  });

  it('handles mentions at word boundaries only', () => {
    const mentionMap = new Map([['test', 'user1']]);
    // @test at word boundary should convert, but not contest@test
    const result = convertMentionsForStorage('hello @test there', mentionMap);
    expect(result).toBe('hello <@user1> there');
  });
});

describe('parseStoredMentions', () => {
  it('parses <@userId> as user mention', () => {
    const segments = parseStoredMentions('Hello <@user123>');
    expect(segments).toEqual([
      { type: 'text', content: 'Hello ' },
      { type: 'user_mention', content: 'user123' },
    ]);
  });

  it('parses <!here> as special mention', () => {
    const segments = parseStoredMentions('Hey <!here>');
    expect(segments).toEqual([
      { type: 'text', content: 'Hey ' },
      { type: 'special_mention', content: 'here' },
    ]);
  });

  it('parses <!channel> as special mention', () => {
    const segments = parseStoredMentions('<!channel> update');
    expect(segments).toEqual([
      { type: 'special_mention', content: 'channel' },
      { type: 'text', content: ' update' },
    ]);
  });

  it('returns array of segments for mixed content', () => {
    const segments = parseStoredMentions('Hey <@user1> and <!here>!');
    expect(segments).toEqual([
      { type: 'text', content: 'Hey ' },
      { type: 'user_mention', content: 'user1' },
      { type: 'text', content: ' and ' },
      { type: 'special_mention', content: 'here' },
      { type: 'text', content: '!' },
    ]);
  });

  it('handles text with no mentions', () => {
    const segments = parseStoredMentions('Just plain text');
    expect(segments).toEqual([{ type: 'text', content: 'Just plain text' }]);
  });

  it('handles empty string', () => {
    const segments = parseStoredMentions('');
    expect(segments).toEqual([]);
  });

  it('handles consecutive mentions', () => {
    const segments = parseStoredMentions('<@user1><@user2>');
    expect(segments).toEqual([
      { type: 'user_mention', content: 'user1' },
      { type: 'user_mention', content: 'user2' },
    ]);
  });
});
