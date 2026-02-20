import type { Editor } from '@tiptap/react';
import { useEditorState } from '@tiptap/react';
import { ToggleButton } from 'react-aria-components';
import { tv } from 'tailwind-variants';
import {
  BoldIcon,
  ItalicIcon,
  UnderlineIcon,
  StrikethroughIcon,
  CodeBracketIcon,
  CodeBracketSquareIcon,
  LinkIcon,
  ListBulletIcon,
  NumberedListIcon,
} from '@heroicons/react/24/outline';
import { Tooltip } from '../ui';

function BlockquoteIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      strokeWidth="1.5"
      stroke="currentColor"
      fill="none"
      className={className}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M3.75 4v16m4.5-14h12m-12 6h12m-12 6h8.25"
      />
    </svg>
  );
}

const toolbar = tv({
  slots: {
    container: [
      'flex items-center gap-0.5 px-2 py-1',
      'bg-gray-100 dark:bg-gray-800',
      'rounded-t-lg',
    ],
    button: [
      'p-1.5 rounded transition-colors text-gray-500 dark:text-gray-400 cursor-pointer',
      'hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-200',
      'selected:bg-gray-200 dark:selected:bg-gray-700',
      'selected:text-gray-900 dark:selected:text-white',
      'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
      'disabled:opacity-50 disabled:cursor-not-allowed',
    ],
    separator: ['w-px h-5 bg-gray-200 dark:bg-gray-700 mx-1'],
    icon: 'w-4 h-4',
  },
});

interface ToolbarProps {
  editor: Editor | null;
}

export function Toolbar({ editor }: ToolbarProps) {
  const s = toolbar();

  // Use useEditorState to reactively track active states
  const editorState = useEditorState({
    editor,
    selector: (ctx) => {
      if (!ctx.editor) {
        return null;
      }
      return {
        isBold: ctx.editor.isActive('bold'),
        isItalic: ctx.editor.isActive('italic'),
        isUnderline: ctx.editor.isActive('underline'),
        isStrike: ctx.editor.isActive('strike'),
        isLink: ctx.editor.isActive('link'),
        isOrderedList: ctx.editor.isActive('orderedList'),
        isBulletList: ctx.editor.isActive('bulletList'),
        isBlockquote: ctx.editor.isActive('blockquote'),
        isCode: ctx.editor.isActive('code'),
        isCodeBlock: ctx.editor.isActive('codeBlock'),
      };
    },
  });

  if (!editor) {
    // Render container with spacer matching button height to prevent layout shift
    // while useEditor initializes (immediatelyRender: false)
    return (
      <div className={s.container()}>
        <div className="h-7" />
      </div>
    );
  }

  const addLink = () => {
    const previousUrl = editor.getAttributes('link').href;
    const url = window.prompt('URL', previousUrl);

    if (url === null) {
      return;
    }

    if (url === '') {
      editor.chain().focus().extendMarkRange('link').unsetLink().run();
      return;
    }

    editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
  };

  return (
    <div className={s.container()}>
      {/* Text formatting: Bold, Italic, Underline, Strikethrough */}
      <Tooltip content="Bold (⌘B)">
        <ToggleButton
          isSelected={editorState?.isBold ?? false}
          onPress={() => editor.chain().focus().toggleBold().run()}
          className={s.button()}
          aria-label="Bold (⌘B)"
        >
          <BoldIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Italic (⌘I)">
        <ToggleButton
          isSelected={editorState?.isItalic ?? false}
          onPress={() => editor.chain().focus().toggleItalic().run()}
          className={s.button()}
          aria-label="Italic (⌘I)"
        >
          <ItalicIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Underline (⌘U)">
        <ToggleButton
          isSelected={editorState?.isUnderline ?? false}
          onPress={() => editor.chain().focus().toggleUnderline().run()}
          className={s.button()}
          aria-label="Underline (⌘U)"
        >
          <UnderlineIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Strikethrough (⌘⇧X)">
        <ToggleButton
          isSelected={editorState?.isStrike ?? false}
          onPress={() => editor.chain().focus().toggleStrike().run()}
          className={s.button()}
          aria-label="Strikethrough (⌘⇧X)"
        >
          <StrikethroughIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <div className={s.separator()} />

      {/* Link, OrderedList, BulletList */}
      <Tooltip content="Insert link">
        <ToggleButton
          isSelected={editorState?.isLink ?? false}
          onPress={addLink}
          className={s.button()}
          aria-label="Insert link"
        >
          <LinkIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Numbered list">
        <ToggleButton
          isSelected={editorState?.isOrderedList ?? false}
          onPress={() => editor.chain().focus().toggleOrderedList().run()}
          className={s.button()}
          aria-label="Numbered list"
        >
          <NumberedListIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Bullet list">
        <ToggleButton
          isSelected={editorState?.isBulletList ?? false}
          onPress={() => editor.chain().focus().toggleBulletList().run()}
          className={s.button()}
          aria-label="Bullet list"
        >
          <ListBulletIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <div className={s.separator()} />

      {/* Quote, Code, CodeBlock */}
      <Tooltip content="Blockquote">
        <ToggleButton
          isSelected={editorState?.isBlockquote ?? false}
          onPress={() => editor.chain().focus().toggleBlockquote().run()}
          className={s.button()}
          aria-label="Blockquote"
        >
          <BlockquoteIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Inline code (⌘E)">
        <ToggleButton
          isSelected={editorState?.isCode ?? false}
          onPress={() => editor.chain().focus().toggleCode().run()}
          className={s.button()}
          aria-label="Inline code (⌘E)"
        >
          <CodeBracketIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>

      <Tooltip content="Code block">
        <ToggleButton
          isSelected={editorState?.isCodeBlock ?? false}
          onPress={() => editor.chain().focus().toggleCodeBlock().run()}
          className={s.button()}
          aria-label="Code block"
        >
          <CodeBracketSquareIcon className={s.icon()} />
        </ToggleButton>
      </Tooltip>
    </div>
  );
}
