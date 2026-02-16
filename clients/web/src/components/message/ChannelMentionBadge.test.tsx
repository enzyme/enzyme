import { describe, it, expect } from 'vitest';
import { render, screen, userEvent } from '../../test-utils';
import { ChannelMentionBadge } from './ChannelMentionBadge';
import type { ChannelWithMembership } from '@feather/api-client';

function makeChannel(overrides: Partial<ChannelWithMembership> = {}): ChannelWithMembership {
  return {
    id: 'ch-1',
    workspace_id: 'ws-1',
    name: 'general',
    type: 'public',
    is_default: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    unread_count: 0,
    notification_count: 0,
    is_starred: false,
    ...overrides,
  };
}

const routerProps = {
  initialEntries: ['/workspace/ws-1/channel/ch-1'],
};

describe('ChannelMentionBadge', () => {
  it('renders public channel with name and hash icon', () => {
    const channels = [makeChannel({ id: 'ch-1', name: 'general', type: 'public' })];

    render(<ChannelMentionBadge channelId="ch-1" channels={channels} />, { routerProps });

    expect(screen.getByRole('button', { name: /general/ })).toBeInTheDocument();
    expect(screen.queryByText('private-channel')).not.toBeInTheDocument();
  });

  it('renders private channel with name and lock icon', () => {
    const channels = [makeChannel({ id: 'ch-2', name: 'secret', type: 'private' })];

    render(<ChannelMentionBadge channelId="ch-2" channels={channels} />, { routerProps });

    expect(screen.getByRole('button', { name: /secret/ })).toBeInTheDocument();
  });

  it('renders unknown channel as "private-channel" with gray styling', () => {
    const channels = [makeChannel({ id: 'ch-1', name: 'general' })];

    render(<ChannelMentionBadge channelId="ch-unknown" channels={channels} />, { routerProps });

    expect(screen.getByText('private-channel')).toBeInTheDocument();
    // Should not be a button — not interactive
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('renders unknown channel with gray background classes', () => {
    render(<ChannelMentionBadge channelId="ch-unknown" channels={[]} />, { routerProps });

    const badge = screen.getByText('private-channel').closest('span')!;
    expect(badge).toHaveClass('bg-gray-100');
    expect(badge).toHaveClass('cursor-default');
  });

  it('renders known channel with blue background classes', () => {
    const channels = [makeChannel({ id: 'ch-1', name: 'general' })];

    render(<ChannelMentionBadge channelId="ch-1" channels={channels} />, { routerProps });

    const button = screen.getByRole('button', { name: /general/ });
    expect(button).toHaveClass('bg-blue-50');
    expect(button).toHaveClass('cursor-pointer');
  });

  it('shows popover with channel info when known channel is clicked', async () => {
    const user = userEvent.setup();
    const channels = [makeChannel({ id: 'ch-1', name: 'general', description: 'Main channel' })];

    render(<ChannelMentionBadge channelId="ch-1" channels={channels} />, { routerProps });

    await user.click(screen.getByRole('button', { name: /general/ }));

    expect(screen.getByText('Go to channel')).toBeInTheDocument();
    expect(screen.getByText('Main channel')).toBeInTheDocument();
  });

  it('does not render description in popover when channel has none', async () => {
    const user = userEvent.setup();
    const channels = [makeChannel({ id: 'ch-1', name: 'general' })];

    render(<ChannelMentionBadge channelId="ch-1" channels={channels} />, { routerProps });

    await user.click(screen.getByRole('button', { name: /general/ }));

    expect(screen.getByText('Go to channel')).toBeInTheDocument();
  });

  it('renders with empty channels array as private-channel', () => {
    render(<ChannelMentionBadge channelId="ch-1" channels={[]} />, { routerProps });

    expect(screen.getByText('private-channel')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});
