// Re-export generated types with friendly aliases
export type { components, paths, operations } from '../generated/schema';
import type { components } from '../generated/schema';

// User types
export type User = components['schemas']['User'];
export type UpdateProfileInput = components['schemas']['UpdateProfileInput'];

// Auth input types
export type LoginInput = components['schemas']['LoginInput'];
export type RegisterInput = components['schemas']['RegisterInput'];

// Workspace types
export type Workspace = components['schemas']['Workspace'];
export type WorkspaceSummary = components['schemas']['WorkspaceSummary'];
export type WorkspaceMembership = components['schemas']['WorkspaceMembership'];
export type WorkspaceMemberWithUser = components['schemas']['WorkspaceMemberWithUser'];
export type WorkspaceRole = components['schemas']['WorkspaceRole'];
export type WorkspaceSettings = components['schemas']['WorkspaceSettings'];
export type PermissionLevel = components['schemas']['PermissionLevel'];
export type Invite = components['schemas']['Invite'];
export type WorkspaceNotificationSummary = components['schemas']['WorkspaceNotificationSummary'];
export type CreateWorkspaceInput = components['schemas']['CreateWorkspaceInput'];
export type UpdateWorkspaceInput = components['schemas']['UpdateWorkspaceInput'];
export type CreateInviteInput = components['schemas']['CreateInviteInput'];

// Channel types
export type CreateDMInput = components['schemas']['CreateDMInput'];
export type ConvertGroupDMInput = components['schemas']['ConvertGroupDMInput'];
export type Channel = components['schemas']['Channel'];
export type ChannelWithMembership = components['schemas']['ChannelWithMembership'];
export type ChannelType = components['schemas']['ChannelType'];
export type ChannelRole = components['schemas']['ChannelRole'];
export type ChannelMember = components['schemas']['ChannelMember'];
export type MarkReadResponse = components['schemas']['MarkReadResponse'];
export type ChannelReadEventData = components['schemas']['ChannelReadEventData'];
export type CreateChannelInput = components['schemas']['CreateChannelInput'];
export type UpdateChannelInput = components['schemas']['UpdateChannelInput'];

// Message types
export type Message = components['schemas']['Message'];
export type MessageWithUser = components['schemas']['MessageWithUser'];
export type Reaction = components['schemas']['Reaction'];
export type ReactionSummary = components['schemas']['ReactionSummary'];
export type MessageListResult = components['schemas']['MessageListResult'];
export type Attachment = components['schemas']['Attachment'];
export type LinkPreview = components['schemas']['LinkPreview'];
export type ThreadParticipant = components['schemas']['ThreadParticipant'];
export type SendMessageInput = components['schemas']['SendMessageInput'];
export type ListMessagesInput = components['schemas']['ListMessagesInput'];

// Server types
export type ServerInfo = components['schemas']['ServerInfo'];

// API types
export type ApiErrorBody = components['schemas']['ApiError'];
export type ApiErrorResponse = components['schemas']['ApiErrorResponse'];

// SSE types
export type SSEEventType = components['schemas']['SSEEventType'];
export type SSEEvent = components['schemas']['SSEEvent'];
export type TypingEventData = components['schemas']['TypingEventData'];
export type PresenceStatus = components['schemas']['PresenceStatus'];
export type PresenceData = components['schemas']['PresenceData'];

// Notification types
export type NotifyLevel = components['schemas']['NotifyLevel'];
export type NotificationPreferences = components['schemas']['NotificationPreferences'];
export type ThreadSubscriptionStatus = components['schemas']['ThreadSubscriptionStatus'];
export type NotificationData = components['schemas']['NotificationData'];

// Unreads types
export type UnreadMessage = components['schemas']['UnreadMessage'];
export type UnreadMessagesResult = components['schemas']['UnreadMessagesResult'];

// Search types
export type SearchMessage = components['schemas']['SearchMessage'];
export type SearchMessagesResult = components['schemas']['SearchMessagesResult'];
export type SearchMessagesInput = components['schemas']['SearchMessagesInput'];

// Thread types
export type ThreadMessage = components['schemas']['ThreadMessage'];
export type ThreadListResult = components['schemas']['ThreadListResult'];

// Custom emoji types
export type CustomEmoji = components['schemas']['CustomEmoji'];

// Scheduled message types
export type ScheduledMessage = components['schemas']['ScheduledMessage'];
export type ScheduleMessageInput = components['schemas']['ScheduleMessageInput'];
export type UpdateScheduledMessageInput = components['schemas']['UpdateScheduledMessageInput'];

// File signing types
export type SignedUrl = components['schemas']['SignedUrl'];

// Moderation types
export type Ban = components['schemas']['Ban'];
export type BanWithUser = components['schemas']['BanWithUser'];
export type BanUserInput = components['schemas']['BanUserInput'];
export type BlockWithUser = components['schemas']['BlockWithUser'];
export type ModerationLogEntryWithActor = components['schemas']['ModerationLogEntryWithActor'];
