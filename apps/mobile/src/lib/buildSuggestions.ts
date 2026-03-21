import { fuzzyMatch, searchAllEmojis, SPECIAL_MENTIONS } from '@enzyme/shared';

export interface Suggestion {
  id: string;
  displayText: string;
  token: string;
  icon?: string;
  avatarUser?: { display_name: string; avatar_url?: string | null; id?: string };
}

interface Member {
  user_id: string;
  display_name: string;
  avatar_url?: string | null;
}

interface Channel {
  id: string;
  name: string;
  type: string;
}

const TRIGGER_CHARS = new Set(['@', '#', ':']);

interface Trigger {
  char: string;
  query: string;
}

/**
 * Detect @, #, or : trigger at cursor position.
 * Looks backwards from cursor to find a trigger character at a word boundary.
 */
function detectTrigger(text: string, cursorPos: number): Trigger | null {
  const before = text.slice(0, cursorPos);

  for (let i = before.length - 1; i >= 0; i--) {
    const char = before[i];

    if (char === ' ' || char === '\n') break;

    if (TRIGGER_CHARS.has(char)) {
      if (i === 0 || before[i - 1] === ' ' || before[i - 1] === '\n') {
        return {
          char,
          query: before.slice(i + 1),
        };
      }
      break;
    }
  }

  return null;
}

export function buildSuggestions(
  text: string,
  cursorPosition: number,
  members: Member[] | undefined,
  channels: Channel[] | undefined,
): Suggestion[] {
  const trigger = detectTrigger(text, cursorPosition);
  if (!trigger) return [];

  const query = trigger.query.toLowerCase();

  if (trigger.char === '@') {
    const results: Suggestion[] = [];

    for (const special of SPECIAL_MENTIONS) {
      if (!query || fuzzyMatch(query, special.displayName).matches) {
        results.push({
          id: `special-${special.id}`,
          displayText: special.displayName,
          token: `<!${special.id}>`,
          icon: '\u{1F4E2}',
        });
      }
    }

    if (members) {
      for (const member of members) {
        if (!query || fuzzyMatch(query, member.display_name).matches) {
          results.push({
            id: member.user_id,
            displayText: member.display_name,
            token: `<@${member.user_id}>`,
            avatarUser: {
              display_name: member.display_name,
              avatar_url: member.avatar_url,
              id: member.user_id,
            },
          });
        }
      }
    }

    return results.slice(0, 5);
  }

  if (trigger.char === '#') {
    if (!channels) return [];
    const results: Suggestion[] = [];

    for (const ch of channels) {
      if (ch.type === 'dm' || ch.type === 'group_dm') continue;
      if (!query || fuzzyMatch(query, ch.name).matches) {
        results.push({
          id: ch.id,
          displayText: ch.name,
          token: `<#${ch.id}>`,
          icon: '#',
        });
      }
    }

    return results.slice(0, 5);
  }

  if (trigger.char === ':') {
    if (query.length < 2) return [];
    const emojis = searchAllEmojis(query, 5, []);
    return emojis.map((e) => ({
      id: e.shortcode,
      displayText: e.shortcode,
      token: `:${e.shortcode}:`,
      icon: e.emoji,
    }));
  }

  return [];
}
