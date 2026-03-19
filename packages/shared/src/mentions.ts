export interface MentionOption {
  type: 'user' | 'special';
  id: string;
  displayName: string;
  avatarUrl?: string;
  gravatarUrl?: string;
}

export interface ParsedMention {
  type: 'user' | 'special';
  id: string;
  raw: string;
}

export interface MentionTrigger {
  isActive: boolean;
  query: string;
  startIndex: number;
}

export const SPECIAL_MENTIONS: MentionOption[] = [
  { type: 'special', id: 'here', displayName: 'here' },
  { type: 'special', id: 'channel', displayName: 'channel' },
  { type: 'special', id: 'everyone', displayName: 'everyone' },
];

/**
 * Detect @ trigger and extract the query string.
 * Returns null if no active trigger, otherwise returns trigger info.
 */
export function parseMentionTrigger(content: string, cursorPos: number): MentionTrigger | null {
  // Look backwards from cursor to find @ at word boundary
  const textBeforeCursor = content.slice(0, cursorPos);

  // Find the last @ that's at a word boundary (start of string, after space, or after newline)
  let atIndex = -1;
  for (let i = textBeforeCursor.length - 1; i >= 0; i--) {
    const char = textBeforeCursor[i];

    // If we hit a space or newline before finding @, there's no active trigger
    if (char === ' ' || char === '\n') {
      break;
    }

    if (char === '@') {
      // Check if @ is at word boundary (start of string, after space, or after newline)
      if (i === 0 || textBeforeCursor[i - 1] === ' ' || textBeforeCursor[i - 1] === '\n') {
        atIndex = i;
      }
      break;
    }
  }

  if (atIndex === -1) {
    return null;
  }

  const query = textBeforeCursor.slice(atIndex + 1);

  return {
    isActive: true,
    query,
    startIndex: atIndex,
  };
}

/**
 * Insert a mention by replacing the trigger text with the display name.
 * Returns the new content and new cursor position.
 */
export function insertMention(
  content: string,
  trigger: MentionTrigger,
  mention: MentionOption,
): { content: string; cursorPos: number } {
  const before = content.slice(0, trigger.startIndex);
  const after = content.slice(trigger.startIndex + trigger.query.length + 1); // +1 for @

  const mentionText = `@${mention.displayName} `;
  const newContent = before + mentionText + after;
  const newCursorPos = before.length + mentionText.length;

  return { content: newContent, cursorPos: newCursorPos };
}

/**
 * Convert display name mentions to ID-based format for storage.
 * @param content - The message content with @DisplayName mentions
 * @param mentionMap - Map of displayName -> userId for user mentions
 * @returns Content with <@userId> for users and <!here>, <!channel>, <!everyone> for special
 */
export function convertMentionsForStorage(
  content: string,
  mentionMap: Map<string, string>,
): string {
  let result = content;

  // Convert user mentions
  for (const [displayName, userId] of mentionMap) {
    // Match @displayName followed by space or end of string
    const regex = new RegExp(`@${escapeRegExp(displayName)}(?=\\s|$)`, 'g');
    result = result.replace(regex, `<@${userId}>`);
  }

  // Convert special mentions
  for (const special of SPECIAL_MENTIONS) {
    const regex = new RegExp(`@${special.displayName}(?=\\s|$)`, 'g');
    result = result.replace(regex, `<!${special.id}>`);
  }

  return result;
}

function escapeRegExp(string: string): string {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Parse stored message content and extract mentions.
 * Returns an array of segments (text or mention) for rendering.
 */
export interface MessageSegment {
  type: 'text' | 'user_mention' | 'special_mention';
  content: string; // For text: the text content. For mentions: the user ID or special ID
}

export function parseStoredMentions(content: string): MessageSegment[] {
  const segments: MessageSegment[] = [];
  // Match <@userId> for user mentions and <!special> for special mentions
  const mentionRegex = /<(@|!)([^>]+)>/g;

  let lastIndex = 0;
  let match;

  while ((match = mentionRegex.exec(content)) !== null) {
    // Add text before this mention
    if (match.index > lastIndex) {
      segments.push({
        type: 'text',
        content: content.slice(lastIndex, match.index),
      });
    }

    // Add the mention
    const isUser = match[1] === '@';
    segments.push({
      type: isUser ? 'user_mention' : 'special_mention',
      content: match[2], // The ID
    });

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text after last mention
  if (lastIndex < content.length) {
    segments.push({
      type: 'text',
      content: content.slice(lastIndex),
    });
  }

  return segments;
}
