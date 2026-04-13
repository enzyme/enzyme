import { useEffect, useRef, useCallback, useState } from 'react';
import {
  MicrophoneIcon,
  SpeakerXMarkIcon,
  SpeakerWaveIcon,
  PhoneXMarkIcon,
} from '@heroicons/react/24/solid';
import { MicrophoneIcon as MicrophoneOutline } from '@heroicons/react/24/outline';
import type { VoiceParticipant } from '@enzyme/api-client';
import {
  useVoiceParticipants,
  useServerMuteVoice,
  useActiveVoiceChannel,
  useVoiceChannelParticipants,
  useIsUserSpeaking,
  useLocalMuted,
  useLocalDeafened,
  setActiveVoiceChannel,
  setLocalMuted,
  setLocalDeafened,
  setVoiceSpeaking,
  clearVoiceState,
  setVoiceSignalingCallbacks,
  clearVoiceSignalingCallbacks,
} from '@enzyme/shared';
import { useAuth } from '../../hooks';
import { Avatar, Button, UnstyledButton, Spinner, toast, Menu, MenuItem } from '../ui';
import { VoiceClient } from '../../lib/voiceClient';

interface VoiceChannelViewProps {
  channelId: string;
  workspaceRole?: string;
}

function ParticipantTile({
  participant,
  isSelf,
}: {
  participant: VoiceParticipant;
  isSelf: boolean;
}) {
  const speaking = useIsUserSpeaking(participant.user_id);
  const isMuted = participant.is_muted || participant.is_server_muted;

  return (
    <div
      className={`relative flex flex-col items-center gap-2 rounded-xl p-4 transition-all ${
        speaking
          ? 'bg-green-50 ring-2 ring-green-400 dark:bg-green-900/20 dark:ring-green-500'
          : 'bg-gray-50 dark:bg-gray-800'
      }`}
    >
      <div className="relative">
        <Avatar
          src={participant.avatar_url}
          name={participant.display_name || 'User'}
          id={participant.user_id}
          size="lg"
        />
        {isMuted && (
          <div className="absolute -right-1 -bottom-1 rounded-full bg-red-500 p-1">
            <MicrophoneOutline className="h-3 w-3 text-white" />
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="h-[1px] w-4 rotate-45 bg-white" />
            </div>
          </div>
        )}
        {participant.is_deafened && (
          <div className="absolute -bottom-1 -left-1 rounded-full bg-red-500 p-1">
            <SpeakerXMarkIcon className="h-3 w-3 text-white" />
          </div>
        )}
      </div>
      <span className="max-w-[120px] truncate text-sm font-medium text-gray-900 dark:text-white">
        {participant.display_name || 'User'}
        {isSelf && ' (you)'}
      </span>
      {participant.is_server_muted && <span className="text-xs text-red-500">Server muted</span>}
    </div>
  );
}

export function VoiceChannelView({ channelId, workspaceRole }: VoiceChannelViewProps) {
  const { user } = useAuth();
  const voiceClientRef = useRef<VoiceClient | null>(null);
  const [isJoining, setIsJoining] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const activeChannel = useActiveVoiceChannel();
  const isInThisChannel = activeChannel === channelId;
  const localMuted = useLocalMuted();
  const localDeafened = useLocalDeafened();

  // Fetch participants from server (initial state)
  const { data: participantsData } = useVoiceParticipants(channelId);
  // Real-time participant state from voice store
  const storeParticipants = useVoiceChannelParticipants(channelId);
  const serverMute = useServerMuteVoice();

  const isAdmin = workspaceRole === 'admin' || workspaceRole === 'owner';
  const userId = user?.id;

  // Merge server data with real-time store: use server data for display info,
  // store data for mute/speaking state
  const participants: VoiceParticipant[] = (participantsData || []).map((p) => {
    const storeP = storeParticipants.find((sp) => sp.userId === p.user_id);
    if (storeP) {
      return {
        ...p,
        is_muted: storeP.muted,
        is_deafened: storeP.deafened,
        is_server_muted: storeP.serverMuted,
      };
    }
    return p;
  });

  const handleJoin = useCallback(async () => {
    if (!channelId || voiceClientRef.current) return;

    setIsJoining(true);
    setError(null);

    const client = new VoiceClient(channelId, {
      onConnectionStateChange: (state) => {
        if (state === 'failed') {
          setError('Connection failed');
        }
      },
      onLocalSpeakingChange: (speaking) => {
        if (userId) {
          setVoiceSpeaking(userId, speaking);
        }
      },
    });

    voiceClientRef.current = client;

    // Wire SSE signaling events to the voice client
    setVoiceSignalingCallbacks({
      onOffer: (data) => client.handleRemoteOffer(data),
      onICECandidate: (data) => client.handleRemoteICECandidate(data),
    });

    try {
      await client.join();
      setActiveVoiceChannel(channelId);
      setIsJoining(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to join voice channel');
      setIsJoining(false);
      client.leave().catch(() => {}); // cleanup media & peer connection
      voiceClientRef.current = null;
      clearVoiceSignalingCallbacks();
    }
  }, [channelId, userId]);

  const handleLeave = useCallback(async () => {
    const client = voiceClientRef.current;
    if (!client) return;

    await client.leave();
    voiceClientRef.current = null;
    clearVoiceSignalingCallbacks();
    clearVoiceState();
  }, []);

  const handleToggleMute = useCallback(() => {
    const client = voiceClientRef.current;
    if (!client) return;

    const newMuted = !localMuted;
    setLocalMuted(newMuted);
    client.setMuted(newMuted);
  }, [localMuted]);

  const handleToggleDeafen = useCallback(() => {
    const client = voiceClientRef.current;
    if (!client) return;

    const newDeafened = !localDeafened;
    setLocalDeafened(newDeafened);
    client.setDeafened(newDeafened);
    // Deafening also mutes
    if (newDeafened) {
      client.setMuted(true);
    }
  }, [localDeafened]);

  const handleServerMute = useCallback(
    (targetUserId: string, muted: boolean) => {
      serverMute.mutate(
        { channelId, userId: targetUserId, muted },
        {
          onError: () => toast('Failed to server mute user', 'error'),
        },
      );
    },
    [channelId, serverMute],
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      const client = voiceClientRef.current;
      if (client) {
        client.leave();
        voiceClientRef.current = null;
        clearVoiceSignalingCallbacks();
        clearVoiceState();
      }
    };
  }, []);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Participant grid */}
      <div className="flex flex-1 flex-col items-center justify-center gap-8 overflow-y-auto p-8">
        {participants.length === 0 && !isInThisChannel ? (
          <p className="text-gray-500 dark:text-gray-400">No one is in this voice channel yet.</p>
        ) : (
          <div className="flex flex-wrap justify-center gap-4">
            {participants.map((p) => (
              <div key={p.user_id}>
                {isAdmin && p.user_id !== user?.id && isInThisChannel ? (
                  <Menu
                    trigger={
                      <UnstyledButton className="cursor-context-menu rounded-xl focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500">
                        <ParticipantTile participant={p} isSelf={p.user_id === user?.id} />
                      </UnstyledButton>
                    }
                    placement="bottom"
                    align="start"
                  >
                    <MenuItem onAction={() => handleServerMute(p.user_id, !p.is_server_muted)}>
                      {p.is_server_muted ? 'Remove server mute' : 'Server mute'}
                    </MenuItem>
                  </Menu>
                ) : (
                  <ParticipantTile participant={p} isSelf={p.user_id === user?.id} />
                )}
              </div>
            ))}
          </div>
        )}

        {error && <p className="text-sm text-red-500">{error}</p>}

        {/* Controls */}
        <div className="flex items-center gap-3">
          {!isInThisChannel ? (
            <Button onPress={handleJoin} isDisabled={isJoining} className="gap-2">
              {isJoining ? (
                <>
                  <Spinner size="sm" />
                  Connecting...
                </>
              ) : (
                <>
                  <MicrophoneIcon className="h-4 w-4" />
                  Join Voice
                </>
              )}
            </Button>
          ) : (
            <>
              <Button
                onPress={handleToggleMute}
                variant={localMuted ? 'danger' : 'secondary'}
                aria-label={localMuted ? 'Unmute' : 'Mute'}
                className="gap-2"
              >
                {localMuted ? (
                  <>
                    <MicrophoneOutline className="h-4 w-4" />
                    <div className="relative -ml-5 h-4 w-4">
                      <div className="absolute top-1/2 left-1/2 h-[1.5px] w-4 -translate-x-1/2 -translate-y-1/2 rotate-45 bg-current" />
                    </div>
                    Muted
                  </>
                ) : (
                  <>
                    <MicrophoneIcon className="h-4 w-4" />
                    Mute
                  </>
                )}
              </Button>

              <Button
                onPress={handleToggleDeafen}
                variant={localDeafened ? 'danger' : 'secondary'}
                aria-label={localDeafened ? 'Undeafen' : 'Deafen'}
                className="gap-2"
              >
                {localDeafened ? (
                  <>
                    <SpeakerXMarkIcon className="h-4 w-4" />
                    Deafened
                  </>
                ) : (
                  <>
                    <SpeakerWaveIcon className="h-4 w-4" />
                    Deafen
                  </>
                )}
              </Button>

              <Button
                onPress={handleLeave}
                variant="danger"
                aria-label="Disconnect"
                className="gap-2"
              >
                <PhoneXMarkIcon className="h-4 w-4" />
                Disconnect
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
