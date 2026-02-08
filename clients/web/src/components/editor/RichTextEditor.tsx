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
} from '@heroicons/react/24/outline';
import { Button as AriaButton, FileTrigger, Popover, DialogTrigger } from 'react-aria-components';
import { Toolbar } from './Toolbar';
import { EmojiPicker } from './EmojiPicker';
import { UserMention, SpecialMention, ChannelMention } from './extensions';
import { createMentionSuggestion, createEmojiSuggestion, createChannelSuggestion } from './suggestions';
import type { EmojiOption } from './suggestions';
import type { ChannelOption } from './suggestions';
import { toMrkdwn } from './serialization';
import { fromMrkdwn } from './serialization';
import type { MentionOption } from '../../lib/mentions';
import { SPECIAL_MENTIONS } from '../../lib/mentions';
import { cn } from '../../lib/utils';

const editorStyles = tv({
  slots: {
    container: [
      'border border-gray-300 dark:border-gray-700 rounded-lg',
      'bg-white dark:bg-gray-900',
      'focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-primary-500',
      'transition-shadow',
    ],
    content: [
      'px-4 py-3 min-h-[2.5rem] max-h-32 overflow-y-auto',
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
      '[&_.mention]:bg-blue-50',
      '[&_.mention]:dark:bg-blue-900/30',
      '[&_.mention]:rounded',
      '[&_.mention]:px-0.5',
      // Code styles
      '[&_.ProseMirror_code]:bg-gray-100',
      '[&_.ProseMirror_code]:dark:bg-gray-700',
      '[&_.ProseMirror_code]:px-1',
      '[&_.ProseMirror_code]:py-0.5',
      '[&_.ProseMirror_code]:rounded',
      '[&_.ProseMirror_code]:text-sm',
      '[&_.ProseMirror_code]:font-mono',
      '[&_.ProseMirror_code]:text-pink-600',
      '[&_.ProseMirror_code]:dark:text-pink-400',
      // Code block styles
      '[&_.ProseMirror_pre]:bg-gray-100',
      '[&_.ProseMirror_pre]:dark:bg-gray-800',
      '[&_.ProseMirror_pre]:p-3',
      '[&_.ProseMirror_pre]:rounded-lg',
      '[&_.ProseMirror_pre]:overflow-x-auto',
      '[&_.ProseMirror_pre_code]:bg-transparent',
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
      '[&_.ProseMirror_a]:text-primary-600',
      '[&_.ProseMirror_a]:dark:text-primary-400',
      '[&_.ProseMirror_a]:underline',
    ],
    actionRow: [
      'flex items-center justify-between px-2 py-1',
      'bg-white dark:bg-gray-900 rounded-b-lg',
    ],
    actionButton: [
      'p-1.5 rounded transition-colors',
      'text-gray-500 dark:text-gray-400',
      'hover:text-gray-700 dark:hover:text-gray-200',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'disabled:opacity-50 disabled:cursor-not-allowed',
    ],
    sendButton: [
      'p-1.5 rounded transition-colors',
    ],
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
  }>;
  workspaceChannels?: Array<{
    id: string;
    name: string;
    type: 'public' | 'private' | 'dm';
  }>;
  showToolbar?: boolean;
  showActionRow?: boolean;
  disabled?: boolean;
  isPending?: boolean;
  onAttachmentClick?: (files: File[]) => void;
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
    },
    ref
  ) => {
    const s = editorStyles();
    const submittingRef = useRef(false);
    const [emojiPickerOpen, setEmojiPickerOpen] = useState(false);

    // Use refs so suggestion closures (captured by TipTap on mount) always see latest data
    const membersRef = useRef(workspaceMembers);
    const channelsRef = useRef(workspaceChannels);
    useEffect(() => { membersRef.current = workspaceMembers; }, [workspaceMembers]);
    useEffect(() => { channelsRef.current = workspaceChannels; }, [workspaceChannels]);

    // Stable suggestion configs — refs are only read at query time (user interaction),
    // not during render, so the lint warning is a false positive here.
    /* eslint-disable react-hooks/refs */
    const mentionSuggestion = useMemo(
      () => createMentionSuggestion((query: string): MentionOption[] => {
        const memberOptions: MentionOption[] = membersRef.current.map((member) => ({
          type: 'user' as const,
          id: member.user_id,
          displayName: member.display_name,
          avatarUrl: member.avatar_url,
        }));

        const allOptions = [...memberOptions, ...SPECIAL_MENTIONS];
        const lowerQuery = query.toLowerCase();

        const filtered = allOptions.filter((option) =>
          option.displayName.toLowerCase().includes(lowerQuery)
        );

        return filtered.sort((a, b) => {
          const aStartsWith = a.displayName.toLowerCase().startsWith(lowerQuery);
          const bStartsWith = b.displayName.toLowerCase().startsWith(lowerQuery);

          if (aStartsWith && !bStartsWith) return -1;
          if (!aStartsWith && bStartsWith) return 1;

          return a.displayName.localeCompare(b.displayName);
        });
      }),
      []
    );

    const emojiSuggestion = useMemo(() => createEmojiSuggestion(), []);

    const channelSuggestion = useMemo(
      () => createChannelSuggestion((query: string): ChannelOption[] => {
        const channels = channelsRef.current
          .filter(c => c.type !== 'dm')
          .map(c => ({
            id: c.id,
            name: c.name,
            type: c.type as 'public' | 'private',
          }));

        const lowerQuery = query.toLowerCase();
        const filtered = channels.filter((c) =>
          c.name.toLowerCase().includes(lowerQuery)
        );

        return filtered.sort((a, b) => {
          const aStartsWith = a.name.toLowerCase().startsWith(lowerQuery);
          const bStartsWith = b.name.toLowerCase().startsWith(lowerQuery);

          if (aStartsWith && !bStartsWith) return -1;
          if (!aStartsWith && bStartsWith) return 1;

          return a.name.localeCompare(b.name);
        });
      }),
      []
    );
    /* eslint-enable react-hooks/refs */

    const editor = useEditor({
      immediatelyRender: false,
      extensions: [
        StarterKit.configure({
          heading: false,
          horizontalRule: false,
          trailingNode: false,
        }),
        Placeholder.configure({
          placeholder,
        }),
        Link.configure({
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
                .insertContent(emojiProps.emoji)
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
                  const isEmpty = listItem.childCount === 1 &&
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
            const textBefore = doc.textBetween(
              Math.max(0, selection.from - 2),
              selection.from
            );

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
                  .insertContent([
                    { type: 'paragraph' },
                    { type: 'codeBlock' },
                  ])
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

          if (event.key === 'Enter' && !event.shiftKey) {
            const { state } = view;
            const { selection } = state;
            const text = state.doc.textBetween(
              Math.max(0, selection.from - 20),
              selection.from
            );

            // Check if we're typing a mention, emoji, or channel
            const hasPendingMention = text.includes('@') && !text.includes(' ');
            const hasPendingEmoji = text.includes(':') && !text.includes(' ');
            const hasPendingChannel = text.includes('#') && !text.includes(' ');
            if (hasPendingMention || hasPendingEmoji || hasPendingChannel) {
              return false;
            }

            event.preventDefault();
            handleSubmit();
            return true;
          }

          return false;
        },
      },
      onUpdate: () => {
        onTyping?.();
      },
      onBlur: () => {
        onBlur?.();
      },
    });

    const handleSubmit = () => {
      if (!editor || submittingRef.current) return;

      const isEmpty = editor.isEmpty;
      if (isEmpty) return;

      submittingRef.current = true;

      const json = editor.getJSON();
      const mrkdwn = toMrkdwn(json);

      if (mrkdwn.trim()) {
        onSubmit(mrkdwn.trim());
        editor.commands.clearContent();
      }

      submittingRef.current = false;
    };

    const handleEmojiSelect = useCallback(
      (emoji: string) => {
        editor?.chain().focus().insertContent(emoji).run();
        setEmojiPickerOpen(false);
      },
      [editor]
    );

    const insertAtSymbol = useCallback(() => {
      editor?.chain().focus().insertContent('@').run();
    }, [editor]);

    const insertHashSymbol = useCallback(() => {
      editor?.chain().focus().insertContent('#').run();
    }, [editor]);

    const insertText = useCallback((text: string) => {
      editor?.chain().focus().insertContent(text).run();
    }, [editor]);

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
      [editor, insertAtSymbol, insertHashSymbol, insertText]
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

    const canSend = !disabled && !isPending;

    return (
      <div className={s.container()}>
        {showToolbar && <Toolbar editor={editor} />}
        <EditorContent editor={editor} className={s.content()} />
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
                  <AriaButton
                    className={s.actionButton()}
                    aria-label="Attach files"
                    isDisabled={disabled}
                  >
                    <PlusIcon className="w-4 h-4" />
                  </AriaButton>
                </FileTrigger>
              )}

              {/* Emoji picker button */}
              <DialogTrigger isOpen={emojiPickerOpen} onOpenChange={setEmojiPickerOpen}>
                <AriaButton
                  className={s.actionButton()}
                  aria-label="Add emoji"
                  isDisabled={disabled}
                >
                  <FaceSmileIcon className="w-4 h-4" />
                </AriaButton>
                <Popover placement="top start" className={s.emojiPopover()}>
                  <EmojiPicker onSelect={handleEmojiSelect} />
                </Popover>
              </DialogTrigger>

              {/* @ mention button */}
              <AriaButton
                className={s.actionButton()}
                aria-label="Mention someone"
                isDisabled={disabled}
                onPress={insertAtSymbol}
              >
                <AtSymbolIcon className="w-4 h-4" />
              </AriaButton>

              {/* # channel mention button */}
              <AriaButton
                className={s.actionButton()}
                aria-label="Mention channel"
                isDisabled={disabled}
                onPress={insertHashSymbol}
              >
                <HashtagIcon className="w-4 h-4" />
              </AriaButton>
            </div>

            {/* Send button */}
            <button
              type="submit"
              disabled={!canSend}
              className={cn(
                s.sendButton(),
                canSend
                  ? 'text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20'
                  : 'text-gray-400 cursor-not-allowed'
              )}
              onClick={handleSubmit}
            >
              <PaperAirplaneIcon className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    );
  }
);

RichTextEditor.displayName = 'RichTextEditor';
