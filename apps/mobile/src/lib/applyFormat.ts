export interface FormatAction {
  label: string;
  prefix: string;
  suffix: string;
}

export const FORMAT_ACTIONS: FormatAction[] = [
  { label: 'B', prefix: '*', suffix: '*' },
  { label: 'I', prefix: '_', suffix: '_' },
  { label: 'S', prefix: '~', suffix: '~' },
  { label: '<>', prefix: '`', suffix: '`' },
  { label: '>', prefix: '> ', suffix: '' },
  { label: '•', prefix: '• ', suffix: '' },
];

export function applyFormat(
  text: string,
  selection: { start: number; end: number },
  action: FormatAction,
): string {
  const before = text.slice(0, selection.start);
  const selected = text.slice(selection.start, selection.end);
  const after = text.slice(selection.end);
  return `${before}${action.prefix}${selected}${action.suffix}${after}`;
}

const TRIGGER_CHARS = new Set(['@', '#', ':']);

/**
 * Find the active trigger character by scanning backwards from cursor,
 * matching the same word-boundary logic as buildSuggestions' detectTrigger.
 */
function findTriggerIndex(text: string, cursorPosition: number): number {
  const before = text.slice(0, cursorPosition);

  for (let i = before.length - 1; i >= 0; i--) {
    const char = before[i];
    if (char === ' ' || char === '\n') break;
    if (TRIGGER_CHARS.has(char)) {
      if (i === 0 || before[i - 1] === ' ' || before[i - 1] === '\n') {
        return i;
      }
      break;
    }
  }

  return -1;
}

export function insertMentionToken(
  text: string,
  cursorPosition: number,
  token: string,
): { newText: string; newPosition: number } | null {
  const triggerIndex = findTriggerIndex(text, cursorPosition);
  if (triggerIndex === -1) return null;

  const before = text.slice(0, triggerIndex);
  const after = text.slice(cursorPosition);
  const newText = `${before}${token} ${after}`;
  const newPosition = before.length + token.length + 1;
  return { newText, newPosition };
}
