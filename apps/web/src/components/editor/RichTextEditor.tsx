import {
  useRef,
  useEffect,
  useImperativeHandle,
  forwardRef,
  useMemo,
  useCallback,
  useState,
} from 'react';
import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Placeholder from '@tiptap/extension-placeholder';
import Link from '@tiptap/extension-link';
import Underline from '@tiptap/extension-underline';
import Mention from '@tiptap/extension-mention';
import { tv } from 'tailwind-variants';
import {
  PlusIcon,
  FaceSmileIcon,
  AtSymbolIcon,
  HashtagIcon,
  PaperAirplaneIcon,
  ClockIcon,
} from '@heroicons/react/24/outline';
import { Button as AriaButton, FileTrigger, Popover, DialogTrigger } from 'react-aria-components';
import { Menu, MenuItem, DisclosureCaret } from '../ui';
import { Toolbar } from './Toolbar';
import { LinkModal } from './LinkModal';
import type { LinkModalData } from './LinkModal';
import { LinkBubbleMenu } from './LinkBubbleMenu';
import { getLinkRange } from './linkUtils';
import { EmojiPicker } from './EmojiPicker';
import { UserMention, SpecialMention, ChannelMention, EmojiNode } from './extensions';
import {
  createMentionSuggestion,
  createEmojiSuggestion,
  createChannelSuggestion,
} from './suggestions';
import type { EmojiOption } from './suggestions';
import type { ChannelOption } from './suggestions';
import { toMrkdwn } from './serialization';
import { fromMrkdwn } from './serialization';
import type { MentionOption } from '../../lib/mentions';
import { SPECIAL_MENTIONS } from '../../lib/mentions';
import { cn } from '../../lib/utils';
import { Tooltip } from '../ui';
import type { EmojiSelectAttrs } from '../ui';
import type { CustomEmoji } from '@enzyme/api-client';

function getScheduleQuickOptions(): { label: string; date: Date }[] {
  const now = new Date();
  const hour = now.getHours();
  const options: { label: string; date: Date }[] = [];

  // "Later today" - next round hour, if before 5pm
  if (hour < 17) {
    const laterHour = Math.min(hour + 1, 17);
    const laterToday = new Date();
    laterToday.setHours(laterHour, 0, 0, 0);
    if (laterToday > now) {
      const ampm = laterHour >= 12 ? 'PM' : 'AM';
      const h = laterHour % 12 || 12;
      options.push({
        label: `Later today at ${h}:00 ${ampm}`,
        date: laterToday,
      });
    }
  }

  // "Tomorrow at 9:00 AM"
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(9, 0, 0, 0);
  options.push({ label: 'Tomorrow at 9:00 AM', date: tomorrow });

  // "Next Monday at 9:00 AM"
  const nextMonday = new Date();
  const dayOfWeek = nextMonday.getDay();
  const daysUntilMonday = dayOfWeek === 0 ? 1 : dayOfWeek === 1 ? 7 : 8 - dayOfWeek;
  nextMonday.setDate(nextMonday.getDate() + daysUntilMonday);
  nextMonday.setHours(9, 0, 0, 0);
  options.push({ label: 'Next Monday at 9:00 AM', date: nextMonday });

  return options;
}

const editorStyles = tv({
  slots: {
    container: [
      'border border-gray-300 dark:border-gray-700 rounded-lg',
      'bg-white dark:bg-gray-900',
      'focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500',
      'transition-shadow',
    ],
    content: [
      'px-4 py-3 min-h-[3rem] max-h-32 overflow-y-auto',
      'prose prose-sm dark:prose-invert max-w-none',
      // Editor content styles
      '[&_.ProseMirror]:outline-none',
      '[&_.ProseMirror]:min-h-[1.5rem]',
      '[&_.ProseMirror]:text-gray-900',
      '[&_.ProseMirror]:dark:text-gray-100',
      '[&_.ProseMirror_p]:my-0',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:content-[attr(data-placeholder)]',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:text-gray-400',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:dark:text-gray-500',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:float-left',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:h-0',
      '[&_.ProseMirror_p.is-editor-empty:first-child::before]:pointer-events-none',
      // Mention styles
      '[&_.mention]:text-blue-600',
      '[&_.mention]:dark:text-blue-400',
      '[&_.mention]:bg-blue-100',
      '[&_.mention]:dark:bg-blue-900',
      '[&_.mention]:rounded',
      '[&_.mention]:px-0.5',
      // Code styles
      '[&_.ProseMirror_code]:bg-gray-200',
      '[&_.ProseMirror_code]:dark:bg-gray-700',
      '[&_.ProseMirror_code]:px-1',
      '[&_.ProseMirror_code]:py-0.5',
      '[&_.ProseMirror_code]:rounded',
      '[&_.ProseMirror_code]:text-sm',
      '[&_.ProseMirror_code]:font-mono',
      '[&_.ProseMirror_code]:text-pink-600',
      '[&_.ProseMirror_code]:dark:text-pink-400',
      // Code block styles
      '[&_.ProseMirror_pre]:bg-gray-200',
      '[&_.ProseMirror_pre]:dark:bg-gray-700',
      '[&_.ProseMirror_pre]:p-3',
      '[&_.ProseMirror_pre]:rounded-lg',
      '[&_.ProseMirror_pre]:overflow-x-auto',
      '[&_.ProseMirror_pre_code]:bg-transparent',
      '[&_.ProseMirror_pre_code]:dark:bg-transparent',
      '[&_.ProseMirror_pre_code]:p-0',
      '[&_.ProseMirror_pre_code]:text-gray-900',
      '[&_.ProseMirror_pre_code]:dark:text-gray-100',
      // Blockquote styles
      '[&_.ProseMirror_blockquote]:border-l-4',
      '[&_.ProseMirror_blockquote]:border-gray-300',
      '[&_.ProseMirror_blockquote]:dark:border-gray-600',
      '[&_.ProseMirror_blockquote]:pl-4',
      '[&_.ProseMirror_blockquote]:italic',
      // List styles
      '[&_.ProseMirror_ul]:list-disc',
      '[&_.ProseMirror_ul]:pl-5',
      '[&_.ProseMirror_ol]:list-decimal',
      '[&_.ProseMirror_ol]:pl-5',
      // Link styles
      '[&_.ProseMirror_a]:text-blue-600',
      '[&_.ProseMirror_a]:dark:text-blue-400',
      '[&_.ProseMirror_a]:underline',
    ],
    actionRow: [
      'flex items-center justify-between p-1.5',
      'bg-white dark:bg-gray-900 rounded-b-lg',
    ],
    actionButton: [
      'p-1.5 rounded transition-colors cursor-pointer',
      'text-gray-500 dark:text-gray-400',
      'hover:text-gray-900 dark:hover:text-gray-200',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'disabled:opacity-50 disabled:cursor-not-allowed',
    ],
    sendButton: ['p-1.5 rounded transition-colors'],
    emojiPopover: [
      'entering:animate-in entering:fade-in entering:zoom-in-95 entering:duration-150',
      'exiting:animate-out exiting:fade-out exiting:zoom-out-95 exiting:duration-100',
    ],
  },
});

export interface RichTextEditorRef {
  clear: () => void;
  focus: () => void;
  isEmpty: () => boolean;
  getContent: () => string;
  insertAtSymbol: () => void;
  insertHashSymbol: () => void;
  insertText: (text: string) => void;
}

const MAX_MESSAGE_LENGTH = 40_000;

export interface RichTextEditorProps {
  initialContent?: string;
  placeholder?: string;
  maxHeight?: number;
  onSubmit: (content: string) => void;
  onTyping?: () => void;
  onBlur?: () => void;
  workspaceMembers?: Array<{
    user_id: string;
    display_name: string;
    avatar_url?: string;
    gravatar_url?: string;
  }>;
  workspaceChannels?: Array<{
    id: string;
    name: string;
    type: 'public' | 'private' | 'dm' | 'group_dm';
  }>;
  showToolbar?: boolean;
  showActionRow?: boolean;
  disabled?: boolean;
  isPending?: boolean;
  onAttachmentClick?: (files: File[]) => void;
  customEmojis?: CustomEmoji[];
  onAddEmoji?: () => void;
  belowEditor?: React.ReactNode;
  onEscape?: () => void;
  submitLabel?: string;
  onScheduleClick?: () => void;
  onSchedule?: (scheduledFor: string) => void;
}

export const RichTextEditor = forwardRef<RichTextEditorRef, RichTextEditorProps>(
  (
    {
      initialContent,
      placeholder = 'Type a message...',
      onSubmit,
      onTyping,
      onBlur,
      workspaceMembers = [],
      workspaceChannels = [],
      showToolbar = true,
      showActionRow = true,
      disabled = false,
      isPending = false,
      onAttachmentClick,
      customEmojis = [],
      onAddEmoji,
      belowEditor,
      onEscape,
      submitLabel,
      onScheduleClick,
      onSchedule,
    },
    ref,
  ) => {
    const s = editorStyles();
    const submittingRef = useRef(false);
    const suggestionOpenRef = useRef(false);
    const onEscapeRef = useRef(onEscape);
    const [emojiPickerOpen, setEmojiPickerOpen] = useState(false);
    const [contentLength, setContentLength] = useState(0);
    const [showLinkModal, setShowLinkModal] = useState(false);
    const [linkModalData, setLinkModalData] = useState<LinkModalData | undefined>();

    useEffect(() => {
      onEscapeRef.current = onEscape;
    }, [onEscape]);

    // Use refs so suggestion closures (captured by TipTap on mount) always see latest data
    const membersRef = useRef(workspaceMembers);
    const channelsRef = useRef(workspaceChannels);
    useEffect(() => {
      membersRef.current = workspaceMembers;
    }, [workspaceMembers]);
    useEffect(() => {
      channelsRef.current = workspaceChannels;
    }, [workspaceChannels]);
    const customEmojisRef = useRef(customEmojis);
    useEffect(() => {
      customEmojisRef.current = customEmojis;
    }, [customEmojis]);

    // Stable suggestion configs — refs are only read at query time (user interaction),
    // not during render, so the lint warning is a false positive here.
    /* eslint-disable react-hooks/refs */
    const handleSuggestionOpenChange = (open: boolean) => {
      suggestionOpenRef.current = open;
    };

    const mentionSuggestion = useMemo(
      () =>
        createMentionSuggestion(
          (query: string): MentionOption[] => {
            const memberOptions: MentionOption[] = membersRef.current.map((member) => ({
              type: 'user' as const,
              id: member.user_id,
              displayName: member.display_name,
              avatarUrl: member.avatar_url,
              gravatarUrl: member.gravatar_url,
            }));

            const allOptions = [...memberOptions, ...SPECIAL_MENTIONS];
            const lowerQuery = query.toLowerCase();

            const filtered = allOptions.filter((option) =>
              option.displayName.toLowerCase().includes(lowerQuery),
            );

            return filtered.sort((a, b) => {
              // Special mentions always go to the bottom
              if (a.type === 'special' && b.type !== 'special') return 1;
              if (a.type !== 'special' && b.type === 'special') return -1;

              const aStartsWith = a.displayName.toLowerCase().startsWith(lowerQuery);
              const bStartsWith = b.displayName.toLowerCase().startsWith(lowerQuery);

              if (aStartsWith && !bStartsWith) return -1;
              if (!aStartsWith && bStartsWith) return 1;

              return a.displayName.localeCompare(b.displayName);
            });
          },
          { onOpenChange: handleSuggestionOpenChange },
        ),
      [],
    );

    const emojiSuggestion = useMemo(
      () => createEmojiSuggestion(customEmojisRef, handleSuggestionOpenChange),
      [],
    );

    const channelSuggestion = useMemo(
      () =>
        createChannelSuggestion((query: string): ChannelOption[] => {
          const channels = channelsRef.current
            .filter((c) => c.type !== 'dm' && c.type !== 'group_dm')
            .map((c) => ({
              id: c.id,
              name: c.name,
              type: c.type as 'public' | 'private',
            }));

          const lowerQuery = query.toLowerCase();
          const filtered = channels.filter((c) => c.name.toLowerCase().includes(lowerQuery));

          return filtered.sort((a, b) => {
            const aStartsWith = a.name.toLowerCase().startsWith(lowerQuery);
            const bStartsWith = b.name.toLowerCase().startsWith(lowerQuery);

            if (aStartsWith && !bStartsWith) return -1;
            if (!aStartsWith && bStartsWith) return 1;

            return a.name.localeCompare(b.name);
          });
        }, handleSuggestionOpenChange),
      [],
    );
    /* eslint-enable react-hooks/refs */

    const editor = useEditor({
      immediatelyRender: false,
      extensions: [
        StarterKit.configure({
          heading: false,
          horizontalRule: false,
          trailingNode: false,
          link: false,
        }),
        Placeholder.configure({
          placeholder,
        }),
        Link.extend({
          inclusive() {
            return false;
          },
        }).configure({
          openOnClick: false,
          HTMLAttributes: {
            rel: 'noopener noreferrer',
            target: '_blank',
          },
        }),
        Underline,
        // User mentions (@)
        Mention.configure({
          HTMLAttributes: {
            class: 'mention',
          },
          suggestion: {
            ...mentionSuggestion,
            command: ({ editor, range, props }) => {
              const mentionProps = props as MentionOption & { label: string };
              const isSpecial = mentionProps.type === 'special';

              editor.chain().focus().deleteRange(range).run();

              if (isSpecial) {
                editor
                  .chain()
                  .focus()
                  .insertContent({
                    type: 'specialMention',
                    attrs: { id: mentionProps.id },
                  })
                  .insertContent(' ')
                  .run();
              } else {
                editor
                  .chain()
                  .focus()
                  .insertContent({
                    type: 'userMention',
                    attrs: {
                      id: mentionProps.id,
                      label: mentionProps.label,
                    },
                  })
                  .insertContent(' ')
                  .run();
              }
            },
          },
        }),
        // Emoji suggestions (:emoji:)
        Mention.extend({ name: 'emojiMention' }).configure({
          suggestion: {
            ...emojiSuggestion,
            command: ({ editor, range, props }) => {
              const emojiProps = props as unknown as EmojiOption;
              editor
                .chain()
                .focus()
                .deleteRange(range)
                .insertContent({
                  type: 'emojiNode',
                  attrs: {
                    shortcode: emojiProps.shortcode,
                    unicode: emojiProps.emoji || null,
                    imageUrl: emojiProps.imageUrl || null,
                  },
                })
                .insertContent(' ')
                .run();
            },
          },
        }),
        // Channel mentions (#)
        Mention.extend({ name: 'channelSuggestion' }).configure({
          suggestion: {
            ...channelSuggestion,
            command: ({ editor, range, props }) => {
              const channelProps = props as unknown as ChannelOption;
              editor.chain().focus().deleteRange(range).run();
              editor
                .chain()
                .focus()
                .insertContent({
                  type: 'channelMention',
                  attrs: {
                    id: channelProps.id,
                    label: channelProps.name,
                  },
                })
                .insertContent(' ')
                .run();
            },
          },
        }),
        UserMention,
        SpecialMention,
        ChannelMention,
        EmojiNode,
      ],
      content: initialContent ? fromMrkdwn(initialContent) : '',
      editable: !disabled,
      editorProps: {
        attributes: {
          class: 'focus:outline-none',
        },
        handleKeyDown: (view, event) => {
          // Handle backspace in empty list item - lift out instead of merging with previous
          if (event.key === 'Backspace') {
            const { state } = view;
            const { $from, empty } = state.selection;

            if (empty && $from.parentOffset === 0) {
              // Check if we're in a list item
              for (let depth = $from.depth; depth > 0; depth--) {
                if ($from.node(depth).type.name === 'listItem') {
                  const listItem = $from.node(depth);
                  // Check if list item is empty (only contains empty paragraph)
                  const isEmpty =
                    listItem.childCount === 1 &&
                    listItem.firstChild?.type.name === 'paragraph' &&
                    listItem.firstChild?.content.size === 0;

                  if (isEmpty) {
                    // Lift out of list instead of merging with previous item
                    editor?.chain().focus().liftListItem('listItem').run();
                    return true;
                  }
                  break;
                }
              }
            }
          }

          // Shift+Enter in lists should create new list item
          if (event.key === 'Enter' && event.shiftKey) {
            const { state } = view;
            const { $from } = state.selection;
            // Check if we're in a list item by walking up the node tree
            for (let depth = $from.depth; depth > 0; depth--) {
              if ($from.node(depth).type.name === 'listItem') {
                // Split the list item instead of creating hard break
                editor?.chain().focus().splitListItem('listItem').run();
                return true;
              }
            }
          }

          // Handle ``` to create code block immediately
          if (event.key === '`') {
            const { state } = view;
            const { selection, doc } = state;
            const { $from } = selection;
            const textBefore = doc.textBetween(Math.max(0, selection.from - 2), selection.from);

            // If we just typed `` and are about to type the third backtick
            if (textBefore === '``') {
              event.preventDefault();

              // Check if there's text before the backticks on this line
              const lineStart = $from.start();
              const textBeforeBackticks = doc.textBetween(lineStart, selection.from - 2);

              if (textBeforeBackticks.trim()) {
                // There's text before - delete backticks, add line break, then code block
                editor
                  ?.chain()
                  .focus()
                  .deleteRange({ from: selection.from - 2, to: selection.from })
                  .insertContent([{ type: 'paragraph' }, { type: 'codeBlock' }])
                  .run();
              } else {
                // Empty/whitespace-only line - select entire paragraph and replace with code block
                const paragraphStart = $from.start();
                const paragraphEnd = $from.end();
                editor
                  ?.chain()
                  .focus()
                  .deleteRange({ from: paragraphStart, to: paragraphEnd })
                  .setCodeBlock()
                  .run();
              }
              return true;
            }
          }

          if (event.key === 'Escape') {
            // If a suggestion popup is open, let TipTap close it first
            if (suggestionOpenRef.current) {
              return false;
            }
            if (onEscapeRef.current) {
              onEscapeRef.current();
              return true;
            }
          }

          if (event.key === 'Enter' && !event.shiftKey) {
            // If a suggestion popup is open, let TipTap's suggestion handle Enter
            if (suggestionOpenRef.current) {
              return false;
            }

            event.preventDefault();
            handleSubmit();
            return true;
          }

          return false;
        },
      },
      onUpdate: ({ editor: e }) => {
        onTyping?.();
        setContentLength([...toMrkdwn(e.getJSON()).trim()].length);
      },
      onBlur: () => {
        onBlur?.();
      },
    });

    const handleSubmit = () => {
      if (!editor || submittingRef.current || disabled || isPending) return;

      const isEmpty = editor.isEmpty;
      if (isEmpty) return;

      if (contentLength > MAX_MESSAGE_LENGTH) return;

      submittingRef.current = true;

      const json = editor.getJSON();
      const mrkdwn = toMrkdwn(json);

      if (mrkdwn.trim()) {
        onSubmit(mrkdwn.trim());
        if (!submitLabel) {
          editor.commands.clearContent();
          setContentLength(0);
        }
      }

      submittingRef.current = false;
    };

    const handleEmojiSelect = useCallback(
      (emoji: string, attrs?: EmojiSelectAttrs) => {
        if (attrs) {
          editor
            ?.chain()
            .focus()
            .insertContent({
              type: 'emojiNode',
              attrs: {
                shortcode: attrs.shortcode,
                unicode: attrs.unicode || null,
                imageUrl: attrs.imageUrl || null,
              },
            })
            .run();
        } else {
          editor?.chain().focus().insertContent(emoji).run();
        }
        setEmojiPickerOpen(false);
      },
      [editor],
    );

    const insertAtSymbol = useCallback(() => {
      editor?.chain().focus().insertContent('@').run();
    }, [editor]);

    const insertHashSymbol = useCallback(() => {
      editor?.chain().focus().insertContent('#').run();
    }, [editor]);

    const insertText = useCallback(
      (text: string) => {
        editor?.chain().focus().insertContent(text).run();
      },
      [editor],
    );

    const openLinkModal = useCallback(() => {
      if (!editor) return;
      const { from, to } = editor.state.selection;
      let selectedText = editor.state.doc.textBetween(from, to, '');
      const existingHref = editor.getAttributes('link').href || '';

      // When cursor is inside a link with no selection, grab the full link text
      if (!selectedText && existingHref) {
        const range = getLinkRange(editor.state, from);
        if (range) {
          selectedText = editor.state.doc.textBetween(range.from, range.to, '');
        }
      }

      setLinkModalData({
        text: selectedText,
        url: existingHref,
        isEdit: !!existingHref,
      });
      setShowLinkModal(true);
    }, [editor]);

    const handleLinkSave = useCallback(
      (text: string, url: string) => {
        if (!editor) return;

        if (text) {
          const { from, to } = editor.state.selection;
          const hasSelection = from !== to;
          const chain = editor.chain().focus();

          if (hasSelection || editor.isActive('link')) {
            // Replace selected text or existing link
            chain.extendMarkRange('link').deleteSelection();
          }

          chain
            .insertContent({
              type: 'text',
              text,
              marks: [{ type: 'link', attrs: { href: url } }],
            })
            .run();
        } else {
          // No display text provided - just set link on current selection or extend mark range
          editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
        }
      },
      [editor],
    );

    useImperativeHandle(
      ref,
      () => ({
        clear: () => {
          editor?.commands.clearContent();
        },
        focus: () => {
          editor?.commands.focus();
        },
        isEmpty: () => {
          return editor?.isEmpty ?? true;
        },
        getContent: () => {
          if (!editor) return '';
          return toMrkdwn(editor.getJSON()).trim();
        },
        insertAtSymbol,
        insertHashSymbol,
        insertText,
      }),
      [editor, insertAtSymbol, insertHashSymbol, insertText],
    );

    useEffect(() => {
      if (editor && initialContent !== undefined) {
        const currentContent = toMrkdwn(editor.getJSON()).trim();
        if (currentContent !== initialContent) {
          editor.commands.setContent(fromMrkdwn(initialContent));
        }
      }
    }, [editor, initialContent]);

    useEffect(() => {
      if (editor) {
        editor.setEditable(!disabled);
      }
    }, [editor, disabled]);

    const isOverLimit = contentLength > MAX_MESSAGE_LENGTH;
    const hasContent = !(editor?.isEmpty ?? true);
    const canSend = !disabled && !isPending && !isOverLimit;
    const canSchedule = canSend && hasContent;

    return (
      <div className={s.container()}>
        {showToolbar && <Toolbar editor={editor} onLinkClick={openLinkModal} />}
        <EditorContent editor={editor} className={s.content()} />
        {showToolbar && <LinkBubbleMenu editor={editor} onEditLink={openLinkModal} />}
        {belowEditor}
        {showActionRow && (
          <div className={s.actionRow()}>
            <div className="flex items-center gap-0.5">
              {/* Attachment button */}
              {onAttachmentClick && (
                <FileTrigger
                  acceptedFileTypes={['image/*', '.pdf', '.txt', '.doc', '.docx']}
                  allowsMultiple
                  onSelect={(files) => files && onAttachmentClick(Array.from(files))}
                >
                  <Tooltip content="Attach files">
                    <AriaButton
                      className={s.actionButton()}
                      aria-label="Attach files"
                      isDisabled={disabled}
                    >
                      <PlusIcon className="h-4 w-4" />
                    </AriaButton>
                  </Tooltip>
                </FileTrigger>
              )}

              {/* Emoji picker button */}
              <DialogTrigger isOpen={emojiPickerOpen} onOpenChange={setEmojiPickerOpen}>
                <Tooltip content="Add emoji">
                  <AriaButton
                    className={s.actionButton()}
                    aria-label="Add emoji"
                    isDisabled={disabled}
                  >
                    <FaceSmileIcon className="h-4 w-4" />
                  </AriaButton>
                </Tooltip>
                <Popover placement="top start" className={s.emojiPopover()}>
                  <EmojiPicker
                    onSelect={handleEmojiSelect}
                    customEmojis={customEmojis}
                    onAddEmoji={
                      onAddEmoji
                        ? () => {
                            setEmojiPickerOpen(false);
                            onAddEmoji();
                          }
                        : undefined
                    }
                  />
                </Popover>
              </DialogTrigger>

              {/* @ mention button */}
              <Tooltip content="Mention someone">
                <AriaButton
                  className={s.actionButton()}
                  aria-label="Mention someone"
                  isDisabled={disabled}
                  onPress={insertAtSymbol}
                >
                  <AtSymbolIcon className="h-4 w-4" />
                </AriaButton>
              </Tooltip>

              {/* # channel mention button */}
              <Tooltip content="Mention channel">
                <AriaButton
                  className={s.actionButton()}
                  aria-label="Mention channel"
                  isDisabled={disabled}
                  onPress={insertHashSymbol}
                >
                  <HashtagIcon className="h-4 w-4" />
                </AriaButton>
              </Tooltip>
            </div>

            <div className="flex items-center gap-1.5">
              {/* Character count (visible when approaching or over limit) */}
              {contentLength > MAX_MESSAGE_LENGTH * 0.9 && (
                <span
                  className={cn(
                    'text-xs tabular-nums',
                    isOverLimit
                      ? 'font-medium text-red-600 dark:text-red-400'
                      : 'text-gray-400 dark:text-gray-500',
                  )}
                >
                  {contentLength.toLocaleString()}/{MAX_MESSAGE_LENGTH.toLocaleString()}
                </span>
              )}

              {/* Send / Save button */}
              {submitLabel ? (
                <div className="flex items-center gap-1.5">
                  {onEscape && (
                    <AriaButton
                      className="cursor-pointer rounded border border-gray-300 px-2 py-1 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
                      onPress={onEscape}
                    >
                      Cancel
                    </AriaButton>
                  )}
                  <AriaButton
                    isDisabled={!canSend}
                    className={cn(
                      'rounded px-2 py-1 text-xs font-medium transition-colors',
                      canSend
                        ? 'cursor-pointer bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600'
                        : 'cursor-not-allowed bg-blue-600/50 text-white/70',
                    )}
                    onPress={handleSubmit}
                  >
                    {isPending ? 'Saving...' : submitLabel}
                  </AriaButton>
                </div>
              ) : (
                <div className="flex items-center">
                  <Tooltip content="Send message">
                    <AriaButton
                      isDisabled={!canSchedule}
                      className={cn(
                        s.sendButton(),
                        onScheduleClick ? 'rounded-r-none' : '',
                        canSchedule
                          ? 'cursor-pointer bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600'
                          : 'cursor-not-allowed bg-gray-100 text-gray-400 dark:bg-gray-800 dark:text-gray-500',
                      )}
                      onPress={handleSubmit}
                    >
                      <PaperAirplaneIcon className="h-4 w-4" />
                    </AriaButton>
                  </Tooltip>
                  {onScheduleClick && (
                    <Menu
                      trigger={
                        <AriaButton
                          isDisabled={!canSchedule}
                          className={cn(
                            'rounded-r px-1.5 py-2 transition-colors',
                            canSchedule
                              ? 'cursor-pointer bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600'
                              : 'cursor-not-allowed bg-gray-100 text-gray-400 dark:bg-gray-800 dark:text-gray-500',
                          )}
                          aria-label="Schedule message"
                        >
                          <DisclosureCaret isExpanded className="h-3 w-3" />
                        </AriaButton>
                      }
                      align="end"
                    >
                      {getScheduleQuickOptions().map((option) => (
                        <MenuItem
                          key={option.label}
                          icon={<ClockIcon className="h-4 w-4" />}
                          onAction={() => onSchedule?.(option.date.toISOString())}
                        >
                          {option.label}
                        </MenuItem>
                      ))}
                      <MenuItem icon={<ClockIcon className="h-4 w-4" />} onAction={onScheduleClick}>
                        Custom time
                      </MenuItem>
                    </Menu>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
        <LinkModal
          isOpen={showLinkModal}
          onClose={() => setShowLinkModal(false)}
          onSave={handleLinkSave}
          initialData={linkModalData}
        />
      </div>
    );
  },
);

RichTextEditor.displayName = 'RichTextEditor';
