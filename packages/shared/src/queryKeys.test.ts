import { describe, it, expect } from 'vitest';
import {
  authKeys,
  userKeys,
  messageKeys,
  threadKeys,
  channelKeys,
  workspaceKeys,
  emojiKeys,
  unreadKeys,
  pinnedMessageKeys,
  scheduledMessageKeys,
  searchKeys,
  serverKeys,
} from './queryKeys';

describe('queryKeys', () => {
  it('authKeys produces correct keys', () => {
    expect(authKeys.all).toEqual(['auth']);
    expect(authKeys.me()).toEqual(['auth', 'me']);
  });

  it('userKeys produces correct keys', () => {
    expect(userKeys.all).toEqual(['user']);
    expect(userKeys.detail('u1')).toEqual(['user', 'u1']);
  });

  it('messageKeys produces correct keys', () => {
    expect(messageKeys.all).toEqual(['messages']);
    expect(messageKeys.list('ch1')).toEqual(['messages', 'ch1']);
    expect(messageKeys.detail('m1')).toEqual(['message', 'm1']);
  });

  it('threadKeys produces correct keys', () => {
    expect(threadKeys.all).toEqual(['thread']);
    expect(threadKeys.detail('m1')).toEqual(['thread', 'm1']);
    expect(threadKeys.subscription('m1')).toEqual(['thread-subscription', 'm1']);
    expect(threadKeys.userThreads('ws1')).toEqual(['user-threads', 'ws1']);
  });

  it('channelKeys produces correct keys', () => {
    expect(channelKeys.all).toEqual(['channels']);
    expect(channelKeys.list('ws1')).toEqual(['channels', 'ws1']);
    expect(channelKeys.members('ch1')).toEqual(['channel', 'ch1', 'members']);
    expect(channelKeys.notifications('ch1')).toEqual(['channel-notifications', 'ch1']);
  });

  it('workspaceKeys produces correct keys', () => {
    expect(workspaceKeys.all).toEqual(['workspace']);
    expect(workspaceKeys.detail('ws1')).toEqual(['workspace', 'ws1']);
    expect(workspaceKeys.members('ws1')).toEqual(['workspace', 'ws1', 'members']);
    expect(workspaceKeys.bans('ws1')).toEqual(['workspace', 'ws1', 'bans']);
    expect(workspaceKeys.blocks('ws1')).toEqual(['workspace', 'ws1', 'blocks']);
    expect(workspaceKeys.moderationLog('ws1')).toEqual(['workspace', 'ws1', 'moderation-log']);
    expect(workspaceKeys.notifications()).toEqual(['workspaces', 'notifications']);
  });

  it('emojiKeys produces correct keys', () => {
    expect(emojiKeys.all).toEqual(['custom-emojis']);
    expect(emojiKeys.list('ws1')).toEqual(['custom-emojis', 'ws1']);
  });

  it('unreadKeys produces correct keys', () => {
    expect(unreadKeys.list('ws1')).toEqual(['unreads', 'ws1']);
  });

  it('pinnedMessageKeys produces correct keys', () => {
    expect(pinnedMessageKeys.all).toEqual(['pinned-messages']);
    expect(pinnedMessageKeys.list('ch1')).toEqual(['pinned-messages', 'ch1']);
  });

  it('scheduledMessageKeys produces correct keys', () => {
    expect(scheduledMessageKeys.all).toEqual(['scheduled-messages']);
    expect(scheduledMessageKeys.list('ws1')).toEqual(['scheduled-messages', 'ws1']);
  });

  it('searchKeys produces correct keys', () => {
    expect(searchKeys.query('ws1', 'hello')).toEqual([
      'search',
      'ws1',
      'hello',
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    ]);
  });

  it('serverKeys produces correct keys', () => {
    expect(serverKeys.info()).toEqual(['server-info']);
  });
});
