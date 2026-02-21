import { useMemo } from 'react';
import type { WorkspaceMemberWithUser, ChannelWithMembership } from '@enzyme/api-client';

export function useEditorMembers(members?: WorkspaceMemberWithUser[]) {
  return useMemo(
    () =>
      members?.map((m) => ({
        user_id: m.user_id,
        display_name: m.display_name,
        avatar_url: m.avatar_url,
        gravatar_url: m.gravatar_url,
      })) || [],
    [members],
  );
}

export function useEditorChannels(channels?: ChannelWithMembership[]) {
  return useMemo(
    () =>
      channels?.map((c) => ({
        id: c.id,
        name: c.name,
        type: c.type,
      })) || [],
    [channels],
  );
}
