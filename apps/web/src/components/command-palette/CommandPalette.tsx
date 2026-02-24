import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Dialog, Modal as AriaModal, ModalOverlay } from 'react-aria-components';
import {
  MagnifyingGlassIcon,
  HashtagIcon,
  LockClosedIcon,
  ChatBubbleLeftRightIcon,
  PlusIcon,
  InboxIcon,
  ChatBubbleLeftEllipsisIcon,
  ClockIcon,
  Cog6ToothIcon,
  UserPlusIcon,
  CommandLineIcon,
} from '@heroicons/react/24/outline';
import { Avatar, AvatarStack } from '../ui';
import { useProfilePanel } from '../../hooks/usePanel';
import { cn } from '../../lib/utils';
import { useCommandPaletteItems, type CommandPaletteItem } from './useCommandPaletteItems';
import type { WorkspaceSettingsTab } from '../settings/WorkspaceSettingsModal';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  onOpenSearch: (query?: string) => void;
  onCreateChannel: () => void;
  onNewDM: () => void;
  onOpenWorkspaceSettings: (workspaceId: string, tab?: WorkspaceSettingsTab) => void;
}

export function CommandPalette({
  isOpen,
  onClose,
  onOpenSearch,
  onCreateChannel,
  onNewDM,
  onOpenWorkspaceSettings,
}: CommandPaletteProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const navigate = useNavigate();
  const { openProfile } = useProfilePanel();
  const listRef = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);

  const rawItems = useCommandPaletteItems(query);

  // Group items by section, then derive flat list for consistent index tracking
  const sections = groupBySection(rawItems);
  const orderedItems = useMemo(() => sections.flatMap((s) => s.sectionItems), [sections]);
  const indexMap = useMemo(() => new Map(orderedItems.map((item, i) => [item, i])), [orderedItems]);

  // Reset state when modal opens
  const [prevIsOpen, setPrevIsOpen] = useState(false);
  if (isOpen && !prevIsOpen) {
    setQuery('');
    setSelectedIndex(0);
  }
  if (isOpen !== prevIsOpen) {
    setPrevIsOpen(isOpen);
  }

  // Clamp selected index when items change (React-recommended sync pattern)
  const [prevItemsLength, setPrevItemsLength] = useState(0);
  if (orderedItems.length !== prevItemsLength) {
    setPrevItemsLength(orderedItems.length);
    if (orderedItems.length > 0 && selectedIndex >= orderedItems.length) {
      setSelectedIndex(orderedItems.length - 1);
    }
  }

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const selected = listRef.current.querySelector('[data-selected="true"]');
    if (selected) {
      selected.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  const executeItem = useCallback(
    (item: CommandPaletteItem) => {
      onClose();

      if (item.type === 'channel' && item.channel) {
        navigate(`/workspaces/${workspaceId}/channels/${item.channel.id}`);
        return;
      }

      if (item.type === 'person' && item.member) {
        openProfile(item.member.user_id);
        return;
      }

      if (item.type === 'action' && item.actionKey) {
        switch (item.actionKey) {
          case 'search':
            onOpenSearch();
            break;
          case 'search-query':
            onOpenSearch(query.trim());
            break;
          case 'create-channel':
            onCreateChannel();
            break;
          case 'new-dm':
            onNewDM();
            break;
          case 'unreads':
            navigate(`/workspaces/${workspaceId}/unreads`);
            break;
          case 'threads':
            navigate(`/workspaces/${workspaceId}/threads`);
            break;
          case 'scheduled':
            navigate(`/workspaces/${workspaceId}/scheduled`);
            break;
          case 'workspace-settings':
            if (workspaceId) onOpenWorkspaceSettings(workspaceId);
            break;
          case 'invite':
            if (workspaceId) onOpenWorkspaceSettings(workspaceId, 'invite');
            break;
        }
      }
    },
    [
      onClose,
      navigate,
      workspaceId,
      openProfile,
      onOpenSearch,
      onCreateChannel,
      onNewDM,
      onOpenWorkspaceSettings,
      query,
    ],
  );

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
      case 'Tab': {
        if (e.key === 'Tab' && e.shiftKey) {
          e.preventDefault();
          setSelectedIndex((prev) => (prev <= 0 ? orderedItems.length - 1 : prev - 1));
          return;
        }
        e.preventDefault();
        setSelectedIndex((prev) => (prev >= orderedItems.length - 1 ? 0 : prev + 1));
        break;
      }
      case 'ArrowUp': {
        e.preventDefault();
        setSelectedIndex((prev) => (prev <= 0 ? orderedItems.length - 1 : prev - 1));
        break;
      }
      case 'Enter': {
        e.preventDefault();
        if (orderedItems[selectedIndex]) {
          executeItem(orderedItems[selectedIndex]);
        }
        break;
      }
    }
  };

  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={(open) => !open && onClose()}
      isDismissable
      className="entering:animate-in entering:fade-in exiting:animate-out exiting:fade-out entering:duration-200 exiting:duration-150 fixed inset-0 z-50 flex items-start justify-center bg-black/50 pt-[15vh]"
    >
      <AriaModal className="entering:animate-in entering:zoom-in-95 exiting:animate-out exiting:zoom-out-95 entering:duration-200 exiting:duration-150 relative mx-4 w-full max-w-xl rounded-lg bg-white shadow-xl dark:bg-gray-800">
        <Dialog className="outline-none">
          <div className="flex flex-col">
            {/* Input */}
            <div className="flex items-center gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
              <MagnifyingGlassIcon className="h-5 w-5 flex-shrink-0 text-gray-400" />
              <input
                type="text"
                placeholder="Go to channel or person..."
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setSelectedIndex(0);
                }}
                onKeyDown={handleKeyDown}
                className="flex-1 bg-transparent text-gray-900 placeholder-gray-400 outline-none dark:text-white"
                autoFocus
              />
            </div>

            {/* Results */}
            <div ref={listRef} role="listbox" className="max-h-[60vh] overflow-y-auto py-1">
              {sections.map(({ section, sectionItems, hideSectionHeader }) => (
                <div key={section}>
                  {!hideSectionHeader && (
                    <div className="px-4 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400">
                      {section}
                    </div>
                  )}
                  {sectionItems.map((item) => {
                    const globalIndex = indexMap.get(item)!;
                    const isSelected = globalIndex === selectedIndex;
                    return (
                      <PaletteItem
                        key={item.id}
                        item={item}
                        isSelected={isSelected}
                        onClick={() => executeItem(item)}
                        onHover={() => setSelectedIndex(globalIndex)}
                      />
                    );
                  })}
                </div>
              ))}
            </div>

            {/* Keyboard hints */}
            <div className="flex items-center gap-3 border-t border-gray-200 px-4 py-2 text-xs text-gray-400 dark:border-gray-700 dark:text-gray-500">
              <span>
                <kbd className="rounded border border-gray-300 px-1 font-sans dark:border-gray-600">
                  &uarr;&darr;
                </kbd>{' '}
                navigate
              </span>
              <span>
                <kbd className="rounded border border-gray-300 px-1 font-sans dark:border-gray-600">
                  &crarr;
                </kbd>{' '}
                select
              </span>
              <span>
                <kbd className="rounded border border-gray-300 px-1 font-sans dark:border-gray-600">
                  esc
                </kbd>{' '}
                close
              </span>
            </div>
          </div>
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}

function PaletteItem({
  item,
  isSelected,
  onClick,
  onHover,
}: {
  item: CommandPaletteItem;
  isSelected: boolean;
  onClick: () => void;
  onHover: () => void;
}) {
  return (
    <div
      data-selected={isSelected}
      className={cn(
        'flex cursor-pointer items-center gap-3 px-4 py-2',
        isSelected
          ? 'bg-blue-50 dark:bg-blue-900/30'
          : 'hover:bg-gray-50 dark:hover:bg-gray-700/50',
      )}
      onClick={onClick}
      onMouseMove={onHover}
      role="option"
      aria-selected={isSelected}
    >
      <ItemIcon item={item} />
      <div className="min-w-0 flex-1">
        <span className="truncate text-sm text-gray-900 dark:text-white">{item.label}</span>
        {item.sublabel && (
          <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">{item.sublabel}</span>
        )}
      </div>
      {item.badge && (
        <span className="rounded-full bg-red-500 px-1.5 py-0.5 text-xs text-white">
          {item.badge}
        </span>
      )}
      {item.shortcut && (
        <span className="text-xs text-gray-400 dark:text-gray-500">{item.shortcut}</span>
      )}
    </div>
  );
}

function ItemIcon({ item }: { item: CommandPaletteItem }) {
  if (item.type === 'channel' && item.channel) {
    const channel = item.channel;

    if (
      channel.type === 'group_dm' &&
      channel.dm_participants &&
      channel.dm_participants.length > 1
    ) {
      const avatarStackUsers = channel.dm_participants.map((p) => ({
        user_id: p.user_id,
        display_name: p.display_name,
        avatar_url: p.avatar_url,
        gravatar_url: p.gravatar_url,
      }));
      return <AvatarStack users={avatarStackUsers} max={2} size="xs" showCount={false} />;
    }

    if ((channel.type === 'dm' || channel.type === 'group_dm') && channel.dm_participants?.[0]) {
      const p = channel.dm_participants[0];
      return (
        <Avatar
          src={p.avatar_url}
          gravatarSrc={p.gravatar_url}
          name={p.display_name}
          id={p.user_id}
          size="xs"
        />
      );
    }
    if (channel.type === 'private') {
      return <LockClosedIcon className="h-4 w-4 text-gray-500" />;
    }
    return <HashtagIcon className="h-4 w-4 text-gray-500" />;
  }

  if (item.type === 'person' && item.member) {
    const m = item.member;
    return (
      <Avatar
        src={m.avatar_url}
        gravatarSrc={m.gravatar_url}
        name={m.display_name_override || m.display_name}
        id={m.user_id}
        size="xs"
      />
    );
  }

  if (item.type === 'action') {
    return <ActionIcon actionKey={item.actionKey} />;
  }

  return <CommandLineIcon className="h-4 w-4 text-gray-500" />;
}

function ActionIcon({ actionKey }: { actionKey?: string }) {
  const className = 'h-4 w-4 text-gray-500';
  switch (actionKey) {
    case 'search':
    case 'search-query':
      return <MagnifyingGlassIcon className={className} />;
    case 'create-channel':
      return <PlusIcon className={className} />;
    case 'new-dm':
      return <ChatBubbleLeftRightIcon className={className} />;
    case 'unreads':
      return <InboxIcon className={className} />;
    case 'threads':
      return <ChatBubbleLeftEllipsisIcon className={className} />;
    case 'scheduled':
      return <ClockIcon className={className} />;
    case 'workspace-settings':
      return <Cog6ToothIcon className={className} />;
    case 'invite':
      return <UserPlusIcon className={className} />;
    default:
      return <CommandLineIcon className={className} />;
  }
}

const SECTION_ORDER = ['Search', 'Recent', 'Channels', 'Direct Messages', 'People', 'Actions'];

interface Section {
  section: string;
  sectionItems: CommandPaletteItem[];
  hideSectionHeader?: boolean;
}

function groupBySection(items: CommandPaletteItem[]): Section[] {
  const buckets = new Map<string, CommandPaletteItem[]>();

  for (const item of items) {
    const bucket = buckets.get(item.section);
    if (bucket) {
      bucket.push(item);
    } else {
      buckets.set(item.section, [item]);
    }
  }

  const result: Section[] = [];
  for (const section of SECTION_ORDER) {
    const sectionItems = buckets.get(section);
    if (sectionItems && sectionItems.length > 0) {
      result.push({
        section,
        sectionItems,
        hideSectionHeader: section === 'Search',
      });
      buckets.delete(section);
    }
  }
  // Any remaining sections not in the predefined order
  for (const [section, sectionItems] of buckets) {
    if (sectionItems.length > 0) {
      result.push({ section, sectionItems });
    }
  }

  return result;
}
