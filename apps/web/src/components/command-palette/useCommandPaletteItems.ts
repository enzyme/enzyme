import { useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { useChannels, useAuth } from '../../hooks';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { fuzzyMatch } from '../../lib/fuzzyMatch';
import { getRecentChannels } from '../../lib/recentChannels';
import type { ChannelWithMembership, WorkspaceMemberWithUser } from '@enzyme/api-client';

export type CommandPaletteItemType = 'channel' | 'person' | 'action';

export interface CommandPaletteItem {
  id: string;
  type: CommandPaletteItemType;
  label: string;
  sublabel?: string;
  section: string;
  score: number;
  /** Channel data for channel/DM items */
  channel?: ChannelWithMembership;
  /** Member data for person items */
  member?: WorkspaceMemberWithUser;
  /** Action key for action items */
  actionKey?: string;
  /** Keyboard shortcut hint */
  shortcut?: string;
  /** Unread/notification count */
  badge?: number;
}

interface ActionDef {
  key: string;
  label: string;
  shortcut?: string;
  adminOnly?: boolean;
}

const MOD_KEY =
  typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? 'Cmd' : 'Ctrl';

const ACTIONS: ActionDef[] = [
  { key: 'search', label: 'Search messages', shortcut: `${MOD_KEY}+Shift+F` },
  { key: 'create-channel', label: 'Create channel' },
  { key: 'new-dm', label: 'New direct message' },
  { key: 'unreads', label: 'All unreads' },
  { key: 'threads', label: 'Threads' },
  { key: 'scheduled', label: 'Scheduled messages' },
  { key: 'workspace-settings', label: 'Workspace settings' },
  { key: 'invite', label: 'Invite people', adminOnly: true },
];

export function useCommandPaletteItems(query: string): CommandPaletteItem[] {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const { data: channelsData } = useChannels(workspaceId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);
  const { user, workspaces } = useAuth();

  const workspaceMembership = workspaces?.find((w) => w.id === workspaceId);
  const isAdminOrOwner =
    workspaceMembership?.role === 'owner' || workspaceMembership?.role === 'admin';

  return useMemo(() => {
    const channels = channelsData?.channels ?? [];
    const members = membersData?.members ?? [];
    const trimmed = query.trim();

    if (!trimmed) {
      return buildEmptyQueryItems(channels, isAdminOrOwner, workspaceId);
    }

    return buildFilteredItems(trimmed, channels, members, user?.id, isAdminOrOwner);
  }, [query, channelsData, membersData, user, isAdminOrOwner, workspaceId]);
}

function buildEmptyQueryItems(
  channels: ChannelWithMembership[],
  isAdminOrOwner: boolean,
  workspaceId: string | undefined,
): CommandPaletteItem[] {
  const items: CommandPaletteItem[] = [];

  // Recent channels
  if (workspaceId) {
    const recentIds = getRecentChannels(workspaceId);
    const channelMap = new Map(channels.map((c) => [c.id, c]));

    for (const id of recentIds) {
      const channel = channelMap.get(id);
      if (!channel || channel.channel_role === undefined) continue;
      items.push(channelToItem(channel, 'Recent', 1000 - items.length));
    }
  }

  // Actions
  for (const action of ACTIONS) {
    if (action.adminOnly && !isAdminOrOwner) continue;
    items.push(actionToItem(action, 'Actions', 0));
  }

  return items;
}

function buildFilteredItems(
  query: string,
  channels: ChannelWithMembership[],
  members: WorkspaceMemberWithUser[],
  currentUserId: string | undefined,
  isAdminOrOwner: boolean,
): CommandPaletteItem[] {
  // Collect items into section buckets, each sorted by score
  const channelItems: CommandPaletteItem[] = [];
  const dmItems: CommandPaletteItem[] = [];
  const peopleItems: CommandPaletteItem[] = [];
  const actionItems: CommandPaletteItem[] = [];

  // Channels & DMs (only joined)
  for (const channel of channels) {
    if (channel.channel_role === undefined) continue;

    const isDM = channel.type === 'dm' || channel.type === 'group_dm';
    const searchText = isDM
      ? (channel.dm_participants?.map((p) => p.display_name).join(' ') ?? channel.name)
      : channel.name;

    const result = fuzzyMatch(query, searchText);
    if (result.matches) {
      const section = isDM ? 'Direct Messages' : 'Channels';
      const item = channelToItem(channel, section, result.score);
      if (isDM) {
        dmItems.push(item);
      } else {
        channelItems.push(item);
      }
    }
  }

  // Collect user IDs already shown as 1:1 DMs to avoid duplicate results
  const dmUserIds = new Set<string>();
  for (const item of dmItems) {
    const participants = item.channel?.dm_participants;
    if (item.channel?.type === 'dm' && participants?.length === 1) {
      dmUserIds.add(participants[0].user_id);
    }
  }

  // People (skip those already shown as 1:1 DMs)
  for (const member of members) {
    if (member.user_id === currentUserId) continue;
    if (dmUserIds.has(member.user_id)) continue;
    const displayName = member.display_name_override || member.display_name;
    const result = fuzzyMatch(query, displayName);
    if (result.matches) {
      peopleItems.push({
        id: `person-${member.user_id}`,
        type: 'person',
        label: displayName,
        section: 'People',
        score: result.score,
        member,
      });
    }
  }

  // Actions
  for (const action of ACTIONS) {
    if (action.adminOnly && !isAdminOrOwner) continue;
    const result = fuzzyMatch(query, action.label);
    if (result.matches) {
      actionItems.push(actionToItem(action, 'Actions', result.score));
    }
  }

  // Sort each bucket by score descending
  const byScore = (a: CommandPaletteItem, b: CommandPaletteItem) => b.score - a.score;
  channelItems.sort(byScore);
  dmItems.sort(byScore);
  peopleItems.sort(byScore);
  actionItems.sort(byScore);

  // "Search for X" suggestion always goes first
  const searchItem: CommandPaletteItem = {
    id: 'action-search-query',
    type: 'action',
    label: `Search messages for "${query}"`,
    section: 'Search',
    score: 0,
    actionKey: 'search-query',
  };

  // Return items grouped by section in a stable order
  return [searchItem, ...channelItems, ...dmItems, ...peopleItems, ...actionItems];
}

function channelToItem(
  channel: ChannelWithMembership,
  section: string,
  score: number,
): CommandPaletteItem {
  const isDM = channel.type === 'dm' || channel.type === 'group_dm';
  const label = isDM
    ? (channel.dm_participants?.map((p) => p.display_name).join(', ') ?? channel.name)
    : channel.name;

  return {
    id: `channel-${channel.id}`,
    type: 'channel',
    label,
    sublabel: isDM ? undefined : channel.type === 'private' ? 'Private channel' : undefined,
    section,
    score,
    channel,
    badge: channel.notification_count > 0 ? channel.notification_count : undefined,
  };
}

function actionToItem(action: ActionDef, section: string, score: number): CommandPaletteItem {
  return {
    id: `action-${action.key}`,
    type: 'action',
    label: action.label,
    section,
    score,
    actionKey: action.key,
    shortcut: action.shortcut,
  };
}
