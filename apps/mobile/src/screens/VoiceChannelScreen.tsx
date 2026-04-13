import { useEffect, useLayoutEffect, useRef, useCallback, useState } from 'react';
import {
  View,
  Text,
  Pressable,
  ActivityIndicator,
  Alert,
  ScrollView,
  AppState,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { VoiceParticipant } from '@enzyme/api-client';
import {
  useVoiceParticipants,
  useServerMuteVoice,
  useAuth,
  useActiveVoiceChannel,
  useVoiceChannelParticipants,
  useIsUserSpeaking,
  useLocalMuted,
  useLocalDeafened,
  setActiveVoiceChannel,
  setLocalMuted,
  setLocalDeafened,
  clearVoiceState,
  setVoiceSignalingCallbacks,
  clearVoiceSignalingCallbacks,
} from '@enzyme/shared';
import type { MainScreenProps } from '../navigation/types';
import { Avatar } from '../components/ui/Avatar';
import { VoiceClient } from '../lib/voiceClient';

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
    <View
      className={`m-2 items-center rounded-xl p-4 ${
        speaking
          ? 'border-2 border-green-400 bg-green-50 dark:border-green-500 dark:bg-green-900/20'
          : 'border-2 border-transparent bg-neutral-100 dark:bg-neutral-800'
      }`}
      style={{ width: 120 }}
    >
      <Avatar
        user={{
          display_name: participant.display_name || 'User',
          avatar_url: participant.avatar_url,
          id: participant.user_id,
        }}
        size="lg"
      />
      <Text className="mt-2 text-sm font-medium text-neutral-900 dark:text-white" numberOfLines={1}>
        {participant.display_name || 'User'}
        {isSelf && ' (you)'}
      </Text>
      {isMuted && (
        <View className="mt-1 flex-row items-center">
          <Ionicons name="mic-off" size={12} color="#ef4444" />
          <Text className="ml-1 text-xs text-red-500">Muted</Text>
        </View>
      )}
      {participant.is_server_muted && (
        <Text className="mt-0.5 text-xs text-red-500">Server muted</Text>
      )}
    </View>
  );
}

export function VoiceChannelScreen({ route, navigation }: MainScreenProps<'VoiceChannel'>) {
  const { workspaceId, channelId, channelName } = route.params;
  const { user, workspaces } = useAuth();
  const voiceClientRef = useRef<VoiceClient | null>(null);
  const [isJoining, setIsJoining] = useState(false);

  const activeChannel = useActiveVoiceChannel();
  const isInThisChannel = activeChannel === channelId;
  const localMuted = useLocalMuted();
  const localDeafened = useLocalDeafened();

  const { data: participantsData } = useVoiceParticipants(channelId);
  const storeParticipants = useVoiceChannelParticipants(channelId);
  const serverMute = useServerMuteVoice();

  const workspaceMembership = workspaces?.find((w) => w.id === workspaceId);
  const isAdmin = workspaceMembership?.role === 'admin' || workspaceMembership?.role === 'owner';

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

  useLayoutEffect(() => {
    navigation.setOptions({
      title: channelName,
      headerRight: () => (
        <Pressable
          onPress={() => navigation.navigate('ChannelDetails', { workspaceId, channelId })}
          className="px-2"
        >
          <Ionicons name="information-circle-outline" size={24} color="#737373" />
        </Pressable>
      ),
    });
  }, [channelName, navigation, workspaceId, channelId]);

  const handleJoin = useCallback(async () => {
    if (!channelId || voiceClientRef.current) return;

    setIsJoining(true);

    const client = new VoiceClient(channelId, {
      onConnectionStateChange: (state) => {
        if (state === 'connected') {
          // connected
        } else if (state === 'disconnected' || state === 'failed' || state === 'closed') {
          // disconnected/failed/closed
          if (state === 'failed') {
            Alert.alert('Voice Error', 'Connection failed');
          }
        }
      },
    });

    voiceClientRef.current = client;

    setVoiceSignalingCallbacks({
      onOffer: (data) => client.handleRemoteOffer(data),
      onICECandidate: (data) => client.handleRemoteICECandidate(data),
    });

    try {
      await client.join();
      setActiveVoiceChannel(channelId);
      setIsJoining(false);
    } catch (err) {
      Alert.alert(
        'Voice Error',
        err instanceof Error ? err.message : 'Failed to join voice channel',
      );
      setIsJoining(false);
      client.leave().catch(() => {}); // cleanup media & peer connection
      voiceClientRef.current = null;
      clearVoiceSignalingCallbacks();
    }
  }, [channelId]);

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
    if (newDeafened) {
      client.setMuted(true);
    }
  }, [localDeafened]);

  const handleParticipantPress = useCallback(
    (participant: VoiceParticipant) => {
      if (!isAdmin || participant.user_id === user?.id || !isInThisChannel) return;

      Alert.alert(participant.display_name || 'User', undefined, [
        {
          text: participant.is_server_muted ? 'Remove Server Mute' : 'Server Mute',
          onPress: () =>
            serverMute.mutate({
              channelId,
              userId: participant.user_id,
              muted: !participant.is_server_muted,
            }),
        },
        { text: 'Cancel', style: 'cancel' },
      ]);
    },
    [isAdmin, user?.id, isInThisChannel, channelId, serverMute],
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

  // Handle app background/foreground transitions — keep voice alive for short
  // background periods, but clean up if the connection dropped while away.
  const appStateRef = useRef(AppState.currentState);
  useEffect(() => {
    const subscription = AppState.addEventListener('change', (nextState) => {
      if (
        appStateRef.current.match(/inactive|background/) &&
        nextState === 'active' &&
        voiceClientRef.current
      ) {
        // Returned to foreground with a voice client — check if still connected.
        // If the peer connection died while backgrounded, clean up.
        if (!voiceClientRef.current.isConnected()) {
          voiceClientRef.current.leave().catch(() => {});
          voiceClientRef.current = null;
          clearVoiceSignalingCallbacks();
          clearVoiceState();
        }
      }
      appStateRef.current = nextState;
    });
    return () => subscription.remove();
  }, []);

  return (
    <View className="flex-1 bg-white dark:bg-neutral-900">
      {/* Participant grid */}
      <ScrollView
        contentContainerClassName="flex-1 items-center justify-center p-4"
        contentContainerStyle={{ flexGrow: 1 }}
      >
        {participants.length === 0 && !isInThisChannel ? (
          <Text className="text-base text-neutral-500 dark:text-neutral-400">
            No one is in this voice channel yet.
          </Text>
        ) : (
          <View className="flex-row flex-wrap justify-center">
            {participants.map((p) => (
              <Pressable
                key={p.user_id}
                onLongPress={() => handleParticipantPress(p)}
                delayLongPress={300}
              >
                <ParticipantTile participant={p} isSelf={p.user_id === user?.id} />
              </Pressable>
            ))}
          </View>
        )}
      </ScrollView>

      {/* Controls */}
      <View className="border-t border-neutral-200 bg-white px-4 py-4 dark:border-neutral-700 dark:bg-neutral-900">
        {!isInThisChannel ? (
          <Pressable
            className="items-center rounded-xl bg-blue-600 px-6 py-3 active:bg-blue-700"
            onPress={handleJoin}
            disabled={isJoining}
          >
            {isJoining ? (
              <ActivityIndicator color="white" />
            ) : (
              <View className="flex-row items-center">
                <Ionicons name="mic" size={20} color="white" />
                <Text className="ml-2 text-base font-semibold text-white">Join Voice</Text>
              </View>
            )}
          </Pressable>
        ) : (
          <View className="flex-row justify-center gap-3">
            <Pressable
              className={`items-center rounded-xl px-4 py-3 ${
                localMuted
                  ? 'bg-red-600 active:bg-red-700'
                  : 'bg-neutral-200 active:bg-neutral-300 dark:bg-neutral-700 dark:active:bg-neutral-600'
              }`}
              onPress={handleToggleMute}
            >
              <Ionicons
                name={localMuted ? 'mic-off' : 'mic'}
                size={20}
                color={localMuted ? 'white' : '#737373'}
              />
              <Text
                className={`mt-1 text-xs ${localMuted ? 'text-white' : 'text-neutral-600 dark:text-neutral-300'}`}
              >
                {localMuted ? 'Muted' : 'Mute'}
              </Text>
            </Pressable>

            <Pressable
              className={`items-center rounded-xl px-4 py-3 ${
                localDeafened
                  ? 'bg-red-600 active:bg-red-700'
                  : 'bg-neutral-200 active:bg-neutral-300 dark:bg-neutral-700 dark:active:bg-neutral-600'
              }`}
              onPress={handleToggleDeafen}
            >
              <Ionicons
                name={localDeafened ? 'volume-mute' : 'volume-high'}
                size={20}
                color={localDeafened ? 'white' : '#737373'}
              />
              <Text
                className={`mt-1 text-xs ${localDeafened ? 'text-white' : 'text-neutral-600 dark:text-neutral-300'}`}
              >
                {localDeafened ? 'Deafened' : 'Deafen'}
              </Text>
            </Pressable>

            <Pressable
              className="items-center rounded-xl bg-red-600 px-4 py-3 active:bg-red-700"
              onPress={handleLeave}
            >
              <Ionicons name="call" size={20} color="white" />
              <Text className="mt-1 text-xs text-white">Leave</Text>
            </Pressable>
          </View>
        )}
      </View>
    </View>
  );
}
