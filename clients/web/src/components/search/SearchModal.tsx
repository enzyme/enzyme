import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  MagnifyingGlassIcon,
  XMarkIcon,
  HashtagIcon,
  LockClosedIcon,
  ChatBubbleLeftRightIcon,
} from '@heroicons/react/24/outline';
import { Dialog, Modal as AriaModal, ModalOverlay } from 'react-aria-components';
import { type DateValue } from '@internationalized/date';
import { useSearch } from '../../hooks/useSearch';
import { useChannels } from '../../hooks/useChannels';
import { useWorkspaceMembers } from '../../hooks/useWorkspaces';
import { DatePicker, Spinner } from '../ui';
import { formatRelativeTime } from '../../lib/utils';
import type { SearchMessage } from '@feather/api-client';

function dateValueToISO(value: DateValue | null, endOfDay?: boolean): string | undefined {
  if (!value) return undefined;
  const d = new Date(value.year, value.month - 1, value.day);
  if (endOfDay) {
    d.setHours(23, 59, 59);
  }
  return d.toISOString();
}

function SearchChannelIcon({ type }: { type: string }) {
  switch (type) {
    case 'private':
      return <LockClosedIcon className="h-4 w-4 text-gray-500" />;
    case 'dm':
    case 'group_dm':
      return <ChatBubbleLeftRightIcon className="h-4 w-4 text-gray-500" />;
    default:
      return <HashtagIcon className="h-4 w-4 text-gray-500" />;
  }
}

interface SearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialChannelId?: string;
}

export function SearchModal({ isOpen, onClose, initialChannelId }: SearchModalProps) {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [inputValue, setInputValue] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [channelFilter, setChannelFilter] = useState(initialChannelId || '');
  const [userFilter, setUserFilter] = useState('');
  const [afterFilter, setAfterFilter] = useState<DateValue | null>(null);
  const [beforeFilter, setBeforeFilter] = useState<DateValue | null>(null);
  const [offset, setOffset] = useState(0);

  const { data: channelsData } = useChannels(workspaceId);
  const { data: membersData } = useWorkspaceMembers(workspaceId);

  const { data, isLoading, isFetching } = useSearch({
    workspaceId: workspaceId || '',
    query: debouncedQuery,
    channelId: channelFilter || undefined,
    userId: userFilter || undefined,
    after: dateValueToISO(afterFilter),
    before: dateValueToISO(beforeFilter, true),
    limit: 20,
    offset,
  });

  // Debounce query input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(inputValue);
      setOffset(0);
    }, 300);
    return () => clearTimeout(timer);
  }, [inputValue]);

  // Reset state when modal opens (React-recommended sync pattern)
  const [prevIsOpen, setPrevIsOpen] = useState(false);
  if (isOpen && !prevIsOpen) {
    setInputValue('');
    setDebouncedQuery('');
    setChannelFilter(initialChannelId || '');
    setUserFilter('');
    setAfterFilter(null);
    setBeforeFilter(null);
    setOffset(0);
  }
  if (isOpen !== prevIsOpen) {
    setPrevIsOpen(isOpen);
  }

  const handleResultClick = useCallback(
    (message: SearchMessage) => {
      onClose();
      navigate(`/workspaces/${workspaceId}/channels/${message.channel_id}?msg=${message.id}`);
    },
    [navigate, workspaceId, onClose],
  );

  const handleLoadMore = () => {
    if (data && data.has_more) {
      setOffset((prev) => prev + 20);
    }
  };

  const channels = channelsData?.channels || [];
  const members = membersData?.members || [];
  const messages = data?.messages || [];
  const totalCount = data?.total_count || 0;

  return (
    <ModalOverlay
      isOpen={isOpen}
      onOpenChange={(open) => !open && onClose()}
      isDismissable
      className="entering:animate-in entering:fade-in entering:duration-200 exiting:animate-out exiting:fade-out exiting:duration-150 fixed inset-0 z-50 flex items-start justify-center bg-black/50 pt-[15vh]"
    >
      <AriaModal className="entering:animate-in entering:zoom-in-95 entering:duration-200 exiting:animate-out exiting:zoom-out-95 exiting:duration-150 relative mx-4 w-full max-w-2xl rounded-lg bg-white shadow-xl dark:bg-gray-800">
        <Dialog className="outline-none">
          <div className="flex flex-col">
            {/* Search input */}
            <div className="flex items-center gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
              <MagnifyingGlassIcon className="h-5 w-5 flex-shrink-0 text-gray-400" />
              <input
                ref={inputRef}
                type="text"
                placeholder="Search messages..."
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                className="flex-1 bg-transparent text-gray-900 placeholder-gray-400 outline-none dark:text-white"
                autoFocus
              />
              {inputValue && (
                <button
                  onClick={() => {
                    setInputValue('');
                    inputRef.current?.focus();
                  }}
                  className="p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <XMarkIcon className="h-4 w-4" />
                </button>
              )}
              {isFetching && <Spinner size="sm" />}
            </div>

            {/* Filters */}
            <div className="flex items-center gap-2 border-b border-gray-200 px-4 py-2 dark:border-gray-700">
              <select
                value={channelFilter}
                onChange={(e) => {
                  setChannelFilter(e.target.value);
                  setOffset(0);
                }}
                className="rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300"
              >
                <option value="">All channels</option>
                {channels
                  .filter((c) => c.channel_role !== undefined)
                  .map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.type === 'dm' || c.type === 'group_dm'
                        ? c.dm_participants?.map((p) => p.display_name).join(', ') || c.name
                        : `#${c.name}`}
                    </option>
                  ))}
              </select>
              <select
                value={userFilter}
                onChange={(e) => {
                  setUserFilter(e.target.value);
                  setOffset(0);
                }}
                className="rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300"
              >
                <option value="">All people</option>
                {members.map((m) => (
                  <option key={m.user_id} value={m.user_id}>
                    {m.display_name_override || m.display_name}
                  </option>
                ))}
              </select>
              <DatePicker
                aria-label="After date"
                value={afterFilter}
                onChange={(value) => {
                  setAfterFilter(value);
                  setOffset(0);
                }}
                maxValue={beforeFilter ?? undefined}
              />
              <DatePicker
                aria-label="Before date"
                value={beforeFilter}
                onChange={(value) => {
                  setBeforeFilter(value);
                  setOffset(0);
                }}
                minValue={afterFilter ?? undefined}
              />
            </div>

            {/* Results */}
            <div className="max-h-[60vh] overflow-y-auto">
              {!debouncedQuery ? (
                <div className="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  Type to search messages across this workspace
                </div>
              ) : isLoading ? (
                <div className="flex items-center justify-center p-8">
                  <Spinner size="lg" />
                </div>
              ) : messages.length === 0 ? (
                <div className="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  No messages found for "{debouncedQuery}"
                </div>
              ) : (
                <>
                  {/* Result count */}
                  <div className="border-b border-gray-100 px-4 py-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">
                    {totalCount} result{totalCount === 1 ? '' : 's'}
                  </div>

                  {messages.map((message) => (
                    <SearchResultItem
                      key={message.id}
                      message={message}
                      onClick={() => handleResultClick(message)}
                    />
                  ))}

                  {data?.has_more && (
                    <div className="p-4 text-center">
                      <button
                        onClick={handleLoadMore}
                        className="text-sm text-primary-600 hover:underline dark:text-primary-400"
                      >
                        Load more results
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </Dialog>
      </AriaModal>
    </ModalOverlay>
  );
}

function SearchResultItem({ message, onClick }: { message: SearchMessage; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="block w-full border-b border-gray-100 p-4 text-left last:border-b-0 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800/50"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 pt-1">
          <SearchChannelIcon type={message.channel_type} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-center gap-2">
            <span className="font-medium text-gray-900 dark:text-gray-100">
              {message.channel_type === 'dm' || message.channel_type === 'group_dm'
                ? message.user_display_name || 'Direct Message'
                : `#${message.channel_name}`}
            </span>
            <span className="text-xs text-gray-500">{formatRelativeTime(message.created_at)}</span>
          </div>
          <div className="flex items-start gap-2">
            {message.user_avatar_url ? (
              <img
                src={message.user_avatar_url}
                alt=""
                className="h-6 w-6 flex-shrink-0 rounded-full"
              />
            ) : (
              <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-gray-300 text-xs font-medium text-gray-600 dark:bg-gray-600 dark:text-gray-300">
                {(message.user_display_name || '?')[0].toUpperCase()}
              </div>
            )}
            <div className="min-w-0">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {message.user_display_name || 'Unknown'}
              </span>
              <p className="line-clamp-2 text-sm text-gray-600 dark:text-gray-400">
                {message.content}
              </p>
            </div>
          </div>
        </div>
      </div>
    </button>
  );
}
