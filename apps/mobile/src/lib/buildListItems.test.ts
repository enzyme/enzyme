import { describe, it, expect } from 'vitest';
import { buildListItems } from './buildListItems';
import type { MessageWithUser } from '@enzyme/api-client';

function makeMessage(
  id: string,
  date: string,
  overrides?: Partial<MessageWithUser>,
): MessageWithUser {
  return {
    id,
    channel_id: 'ch1',
    user_id: 'u1',
    content: `Message ${id}`,
    created_at: `${date}T12:00:00Z`,
    updated_at: `${date}T12:00:00Z`,
    type: 'user',
    reply_count: 0,
    reactions: [],
    attachments: [],
    display_name: 'User',
    ...overrides,
  } as MessageWithUser;
}

describe('buildListItems', () => {
  it('returns empty array for undefined pages', () => {
    expect(buildListItems(undefined)).toEqual([]);
  });

  it('returns empty array for empty pages', () => {
    expect(buildListItems([])).toEqual([]);
  });

  it('returns empty array for pages with no messages', () => {
    expect(buildListItems([{ messages: [] }])).toEqual([]);
  });

  it('creates message items and a trailing date separator for single message', () => {
    const pages = [{ messages: [makeMessage('m1', '2026-03-20')] }];
    const items = buildListItems(pages);
    expect(items).toHaveLength(2);
    expect(items[0]).toEqual({ type: 'message', data: pages[0].messages[0], isGrouped: false });
    expect(items[1]).toEqual({
      type: 'date',
      date: '2026-03-20T12:00:00Z',
      id: 'date-2026-03-20-last',
    });
  });

  it('inserts date separator between messages on different days', () => {
    const pages = [
      {
        messages: [makeMessage('m1', '2026-03-20'), makeMessage('m2', '2026-03-19')],
      },
    ];
    const items = buildListItems(pages);
    // m1 (Mar 20), date separator (Mar 20), m2 (Mar 19), date separator (Mar 19, last)
    expect(items).toHaveLength(4);
    expect(items[0].type).toBe('message');
    expect(items[1]).toEqual({
      type: 'date',
      date: '2026-03-20T12:00:00Z',
      id: 'date-2026-03-20',
    });
    expect(items[2].type).toBe('message');
    expect(items[3]).toEqual({
      type: 'date',
      date: '2026-03-19T12:00:00Z',
      id: 'date-2026-03-19-last',
    });
  });

  it('does not insert separator between messages on same day', () => {
    const pages = [
      {
        messages: [makeMessage('m1', '2026-03-20'), makeMessage('m2', '2026-03-20')],
      },
    ];
    const items = buildListItems(pages);
    // m1, m2, trailing date separator
    expect(items).toHaveLength(3);
    expect(items[0].type).toBe('message');
    expect(items[1].type).toBe('message');
    expect(items[2].type).toBe('date');
  });

  it('flattens multiple pages', () => {
    const pages = [
      { messages: [makeMessage('m1', '2026-03-20')] },
      { messages: [makeMessage('m2', '2026-03-19')] },
    ];
    const items = buildListItems(pages);
    const messageItems = items.filter((i) => i.type === 'message');
    expect(messageItems).toHaveLength(2);
  });

  it('handles three days of messages correctly', () => {
    const pages = [
      {
        messages: [
          makeMessage('m1', '2026-03-20'),
          makeMessage('m2', '2026-03-19'),
          makeMessage('m3', '2026-03-18'),
        ],
      },
    ];
    const items = buildListItems(pages);
    const dateItems = items.filter((i) => i.type === 'date');
    // Two date-change separators + one trailing separator
    expect(dateItems).toHaveLength(3);
  });

  describe('message grouping', () => {
    it('groups consecutive messages from same user within 5 minutes', () => {
      const pages = [
        {
          messages: [
            makeMessage('m1', '2026-03-20', { created_at: '2026-03-20T12:02:00Z' }),
            makeMessage('m2', '2026-03-20', { created_at: '2026-03-20T12:00:00Z' }),
          ],
        },
      ];
      const items = buildListItems(pages);
      const msgItems = items.filter((i) => i.type === 'message');
      expect(msgItems[0].isGrouped).toBe(true);
      expect(msgItems[1].isGrouped).toBe(false); // last message is never grouped
    });

    it('does not group messages from different users', () => {
      const pages = [
        {
          messages: [
            makeMessage('m1', '2026-03-20', {
              user_id: 'u1',
              created_at: '2026-03-20T12:01:00Z',
            }),
            makeMessage('m2', '2026-03-20', {
              user_id: 'u2',
              created_at: '2026-03-20T12:00:00Z',
            }),
          ],
        },
      ];
      const items = buildListItems(pages);
      const msgItems = items.filter((i) => i.type === 'message');
      expect(msgItems[0].isGrouped).toBe(false);
    });

    it('does not group messages more than 5 minutes apart', () => {
      const pages = [
        {
          messages: [
            makeMessage('m1', '2026-03-20', { created_at: '2026-03-20T12:06:00Z' }),
            makeMessage('m2', '2026-03-20', { created_at: '2026-03-20T12:00:00Z' }),
          ],
        },
      ];
      const items = buildListItems(pages);
      const msgItems = items.filter((i) => i.type === 'message');
      expect(msgItems[0].isGrouped).toBe(false);
    });

    it('does not group system messages', () => {
      const pages = [
        {
          messages: [
            makeMessage('m1', '2026-03-20', {
              type: 'system',
              created_at: '2026-03-20T12:01:00Z',
            }),
            makeMessage('m2', '2026-03-20', { created_at: '2026-03-20T12:00:00Z' }),
          ],
        },
      ];
      const items = buildListItems(pages);
      const msgItems = items.filter((i) => i.type === 'message');
      expect(msgItems[0].isGrouped).toBe(false);
    });

    it('does not group across date boundaries', () => {
      const pages = [
        {
          messages: [
            makeMessage('m1', '2026-03-20', { created_at: '2026-03-20T00:01:00Z' }),
            makeMessage('m2', '2026-03-19', { created_at: '2026-03-19T23:59:00Z' }),
          ],
        },
      ];
      const items = buildListItems(pages);
      const msgItems = items.filter((i) => i.type === 'message');
      expect(msgItems[0].isGrouped).toBe(false);
    });
  });
});
