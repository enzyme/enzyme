import { describe, it, expect } from 'vitest';
import { applyFormat, insertMentionToken, FORMAT_ACTIONS } from './applyFormat';

describe('applyFormat', () => {
  it('wraps selected text with bold markers', () => {
    const bold = FORMAT_ACTIONS.find((a) => a.label === 'B')!;
    const result = applyFormat('hello world', { start: 6, end: 11 }, bold);
    expect(result).toBe('hello *world*');
  });

  it('wraps selected text with italic markers', () => {
    const italic = FORMAT_ACTIONS.find((a) => a.label === 'I')!;
    const result = applyFormat('hello world', { start: 6, end: 11 }, italic);
    expect(result).toBe('hello _world_');
  });

  it('wraps selected text with strikethrough markers', () => {
    const strike = FORMAT_ACTIONS.find((a) => a.label === 'S')!;
    const result = applyFormat('hello world', { start: 6, end: 11 }, strike);
    expect(result).toBe('hello ~world~');
  });

  it('wraps selected text with code markers', () => {
    const code = FORMAT_ACTIONS.find((a) => a.label === '<>')!;
    const result = applyFormat('hello world', { start: 6, end: 11 }, code);
    expect(result).toBe('hello `world`');
  });

  it('inserts blockquote prefix (no suffix)', () => {
    const quote = FORMAT_ACTIONS.find((a) => a.label === '>')!;
    const result = applyFormat('hello world', { start: 0, end: 5 }, quote);
    expect(result).toBe('> hello world');
  });

  it('inserts list prefix (no suffix)', () => {
    const list = FORMAT_ACTIONS.find((a) => a.label === '•')!;
    const result = applyFormat('item one', { start: 0, end: 0 }, list);
    expect(result).toBe('• item one');
  });

  it('inserts markers at cursor when no text is selected', () => {
    const bold = FORMAT_ACTIONS.find((a) => a.label === 'B')!;
    const result = applyFormat('hello world', { start: 5, end: 5 }, bold);
    expect(result).toBe('hello** world');
  });

  it('handles empty text', () => {
    const bold = FORMAT_ACTIONS.find((a) => a.label === 'B')!;
    const result = applyFormat('', { start: 0, end: 0 }, bold);
    expect(result).toBe('**');
  });

  it('handles selection in middle of text', () => {
    const code = FORMAT_ACTIONS.find((a) => a.label === '<>')!;
    const result = applyFormat('call myFunc() here', { start: 5, end: 13 }, code);
    expect(result).toBe('call `myFunc()` here');
  });
});

describe('insertMentionToken', () => {
  it('replaces @ trigger with user mention token', () => {
    const result = insertMentionToken('Hello @al', 9, '<@user1>');
    expect(result).toEqual({
      newText: 'Hello <@user1> ',
      newPosition: 15,
    });
  });

  it('replaces # trigger with channel mention token', () => {
    const result = insertMentionToken('#gen', 4, '<#ch1>');
    expect(result).toEqual({
      newText: '<#ch1> ',
      newPosition: 7,
    });
  });

  it('replaces : trigger with emoji token', () => {
    const result = insertMentionToken('hello :smi', 10, ':smile:');
    expect(result).toEqual({
      newText: 'hello :smile: ',
      newPosition: 14,
    });
  });

  it('preserves text after cursor', () => {
    // Cursor is at position 7 (after "@bo"), not at 5 (after "@")
    const result = insertMentionToken('hey @bo and more', 7, '<@bob>');
    expect(result).toEqual({
      newText: 'hey <@bob>  and more',
      newPosition: 11,
    });
  });

  it('returns null when no trigger found', () => {
    const result = insertMentionToken('hello world', 11, '<@user1>');
    expect(result).toBeNull();
  });
});
