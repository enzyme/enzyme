import type { MessageWithUser } from '@enzyme/api-client';

const GROUP_THRESHOLD_MS = 5 * 60 * 1000; // 5 minutes

/**
 * Determines whether a message should be visually grouped with the next message
 * (hiding the avatar and header). Both messages must be from the same user,
 * not system messages, not deleted, on the same day, and within 5 minutes.
 */
export function shouldGroupMessages(
  current: MessageWithUser,
  next: MessageWithUser | undefined,
): boolean {
  if (!next) return false;
  if (current.type === 'system' || next.type === 'system') return false;
  if (current.user_id !== next.user_id) return false;
  if (next.deleted_at) return false;

  const currentDate = current.created_at.split('T')[0];
  const nextDate = next.created_at.split('T')[0];
  if (currentDate !== nextDate) return false;

  return (
    Math.abs(new Date(current.created_at).getTime() - new Date(next.created_at).getTime()) <
    GROUP_THRESHOLD_MS
  );
}

export type ListItem =
  | { type: 'message'; data: MessageWithUser; isGrouped: boolean }
  | { type: 'date'; date: string; id: string };

export function buildListItems(pages: { messages: MessageWithUser[] }[] | undefined): ListItem[] {
  if (!pages) return [];

  const messages = pages.flatMap((p) => p.messages);
  const items: ListItem[] = [];

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    const msgDate = msg.created_at.split('T')[0];
    const prevDate = messages[i + 1]?.created_at.split('T')[0];

    // In an inverted list, items[i+1] is the message visually *above* items[i].
    // A message is "grouped" when the message above it is from the same user within 5 min.
    const isGrouped = shouldGroupMessages(msg, messages[i + 1]);

    items.push({ type: 'message', data: msg, isGrouped });

    if (prevDate && msgDate !== prevDate) {
      items.push({ type: 'date', date: msg.created_at, id: `date-${msgDate}` });
    }
    if (i === messages.length - 1) {
      items.push({ type: 'date', date: msg.created_at, id: `date-${msgDate}-last` });
    }
  }

  return items;
}
