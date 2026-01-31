// Re-export generated types with friendly aliases
export type { components, paths, operations } from '../generated/schema';
import type { components } from '../generated/schema';

// User types
export type User = components['schemas']['User'];

// Workspace types
export type Workspace = components['schemas']['Workspace'];
export type WorkspaceSummary = components['schemas']['WorkspaceSummary'];
export type WorkspaceMembership = components['schemas']['WorkspaceMembership'];
export type WorkspaceMemberWithUser = components['schemas']['WorkspaceMemberWithUser'];
export type WorkspaceRole = components['schemas']['WorkspaceRole'];
export type Invite = components['schemas']['Invite'];

// Channel types
export type Channel = components['schemas']['Channel'];
export type ChannelWithMembership = components['schemas']['ChannelWithMembership'];
export type ChannelType = components['schemas']['ChannelType'];
export type ChannelRole = components['schemas']['ChannelRole'];
export type ChannelMember = components['schemas']['ChannelMember'];

// Message types
export type Message = components['schemas']['Message'];
export type MessageWithUser = components['schemas']['MessageWithUser'];
export type Reaction = components['schemas']['Reaction'];
export type ReactionSummary = components['schemas']['ReactionSummary'];
export type MessageListResult = components['schemas']['MessageListResult'];

// API types
export type ApiError = components['schemas']['ApiError'];
export type ApiErrorResponse = components['schemas']['ApiErrorResponse'];

// SSE types
export type SSEEventType = components['schemas']['SSEEventType'];
export type SSEEvent = components['schemas']['SSEEvent'];
export type TypingEventData = components['schemas']['TypingEventData'];
export type PresenceStatus = components['schemas']['PresenceStatus'];
export type PresenceData = components['schemas']['PresenceData'];
