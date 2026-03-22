import { memo, useMemo } from 'react';
import { Text, View, ScrollView, Linking } from 'react-native';
import {
  parseMrkdwn,
  isEmojiOnly,
  resolveStandardShortcode,
  type MrkdwnSegment,
} from '@enzyme/shared';
import type { WorkspaceMemberWithUser, ChannelWithMembership } from '@enzyme/api-client';

interface MrkdwnRendererProps {
  content: string;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  onMentionPress?: (userId: string) => void;
  onChannelPress?: (channelId: string) => void;
}

export const MrkdwnRenderer = memo(function MrkdwnRenderer({
  content,
  members,
  channels,
  onMentionPress,
  onChannelPress,
}: MrkdwnRendererProps) {
  const segments = useMemo(() => parseMrkdwn(content), [content]);
  const emojiOnly = useMemo(() => isEmojiOnly(segments), [segments]);

  return (
    <Text style={emojiOnly ? { fontSize: 32, lineHeight: 40 } : { fontSize: 16, lineHeight: 22 }}>
      {segments.map((segment, i) => (
        <Segment
          key={i}
          segment={segment}
          members={members}
          channels={channels}
          onMentionPress={onMentionPress}
          onChannelPress={onChannelPress}
        />
      ))}
    </Text>
  );
});

interface SegmentProps {
  segment: MrkdwnSegment;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  onMentionPress?: (userId: string) => void;
  onChannelPress?: (channelId: string) => void;
}

function openSafeURL(url: string) {
  if (/^https?:\/\//i.test(url)) {
    Linking.openURL(url);
  }
}

function Segment({ segment, members, channels, onMentionPress, onChannelPress }: SegmentProps) {
  switch (segment.type) {
    case 'text':
      return <Text>{segment.content}</Text>;

    case 'bold':
      return <Text style={{ fontWeight: 'bold' }}>{segment.content}</Text>;

    case 'italic':
      return <Text style={{ fontStyle: 'italic' }}>{segment.content}</Text>;

    case 'strike':
      return <Text style={{ textDecorationLine: 'line-through' }}>{segment.content}</Text>;

    case 'code':
      return (
        <Text className="rounded bg-neutral-200 px-1 font-mono text-sm text-red-600 dark:bg-neutral-700 dark:text-red-400">
          {segment.content}
        </Text>
      );

    case 'code_block':
      return <CodeBlock content={segment.content} />;

    case 'link':
      return (
        <Text
          className="text-blue-500 underline dark:text-blue-400"
          onPress={() => openSafeURL(segment.url)}
        >
          {segment.text}
        </Text>
      );

    case 'user_mention': {
      const member = members?.find((m) => m.user_id === segment.userId);
      const name = member?.display_name ?? 'Unknown';
      return (
        <Text
          className="rounded bg-blue-100 px-1 font-semibold text-blue-700 dark:bg-blue-900 dark:text-blue-300"
          onPress={() => onMentionPress?.(segment.userId)}
        >
          @{name}
        </Text>
      );
    }

    case 'channel_mention': {
      const channel = channels?.find((c) => c.id === segment.channelId);
      const name = channel?.name ?? 'unknown';
      return (
        <Text
          className="rounded bg-blue-100 px-1 font-semibold text-blue-700 dark:bg-blue-900 dark:text-blue-300"
          onPress={() => onChannelPress?.(segment.channelId)}
        >
          #{name}
        </Text>
      );
    }

    case 'special_mention': {
      const label = `@${segment.mentionType}`;
      return (
        <Text className="rounded bg-yellow-100 px-1 font-semibold text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300">
          {label}
        </Text>
      );
    }

    case 'emoji_shortcode': {
      const emoji = resolveStandardShortcode(segment.name);
      return <Text>{emoji ?? `:${segment.name}:`}</Text>;
    }

    case 'blockquote':
      return (
        <BlockquoteView
          segments={segment.segments}
          members={members}
          channels={channels}
          onMentionPress={onMentionPress}
          onChannelPress={onChannelPress}
        />
      );

    case 'bullet_list':
      return (
        <ListView
          items={segment.items}
          ordered={false}
          members={members}
          channels={channels}
          onMentionPress={onMentionPress}
          onChannelPress={onChannelPress}
        />
      );

    case 'ordered_list':
      return (
        <ListView
          items={segment.items}
          ordered={true}
          members={members}
          channels={channels}
          onMentionPress={onMentionPress}
          onChannelPress={onChannelPress}
        />
      );

    case 'line_break':
      return <Text>{'\n'}</Text>;

    default:
      return null;
  }
}

function CodeBlock({ content }: { content: string }) {
  return (
    <View className="my-1 rounded-md bg-neutral-200 p-2 dark:bg-neutral-700">
      <ScrollView horizontal showsHorizontalScrollIndicator={false}>
        <Text className="font-mono text-sm text-neutral-900 dark:text-neutral-100">{content}</Text>
      </ScrollView>
    </View>
  );
}

interface NestedSegmentProps {
  segments: MrkdwnSegment[];
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  onMentionPress?: (userId: string) => void;
  onChannelPress?: (channelId: string) => void;
}

function BlockquoteView({
  segments,
  members,
  channels,
  onMentionPress,
  onChannelPress,
}: NestedSegmentProps) {
  return (
    <View className="my-1 border-l-4 border-neutral-300 pl-3 dark:border-neutral-600">
      <Text
        className="text-neutral-600 dark:text-neutral-400"
        style={{ fontSize: 16, lineHeight: 22 }}
      >
        {segments.map((s, i) => (
          <Segment
            key={i}
            segment={s}
            members={members}
            channels={channels}
            onMentionPress={onMentionPress}
            onChannelPress={onChannelPress}
          />
        ))}
      </Text>
    </View>
  );
}

interface ListViewProps {
  items: MrkdwnSegment[][];
  ordered: boolean;
  members?: WorkspaceMemberWithUser[];
  channels?: ChannelWithMembership[];
  onMentionPress?: (userId: string) => void;
  onChannelPress?: (channelId: string) => void;
}

function ListView({
  items,
  ordered,
  members,
  channels,
  onMentionPress,
  onChannelPress,
}: ListViewProps) {
  return (
    <View className="my-1">
      {items.map((item, index) => (
        <View key={index} className="flex-row pl-2">
          <Text className="mr-2 text-neutral-500 dark:text-neutral-400" style={{ fontSize: 16 }}>
            {ordered ? `${index + 1}.` : '•'}
          </Text>
          <Text style={{ fontSize: 16, lineHeight: 22, flex: 1 }}>
            {item.map((s, i) => (
              <Segment
                key={i}
                segment={s}
                members={members}
                channels={channels}
                onMentionPress={onMentionPress}
                onChannelPress={onChannelPress}
              />
            ))}
          </Text>
        </View>
      ))}
    </View>
  );
}
