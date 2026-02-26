import { useEffect, useCallback } from 'react';
import type { Editor } from '@tiptap/react';
import { useEditorState } from '@tiptap/react';
import { BubbleMenu } from '@tiptap/react/menus';
import { PencilIcon, TrashIcon, ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import { IconButton } from '../ui';
import { getLinkRange } from './linkUtils';

interface LinkBubbleMenuProps {
  editor: Editor | null;
  onEditLink: () => void;
}

function shouldShow({ editor }: { editor: Editor }) {
  return editor.isActive('link');
}

export function LinkBubbleMenu({ editor, onEditLink }: LinkBubbleMenuProps) {
  const href = useEditorState({
    editor,
    selector: (ctx) => {
      if (!ctx.editor) return undefined;
      const { from } = ctx.editor.state.selection;
      const $pos = ctx.editor.state.doc.resolve(from);
      const linkMark = $pos.marks().find((m) => m.type.name === 'link');
      return (linkMark?.attrs.href as string) || undefined;
    },
  });

  // After React commits the new href content, tell the plugin to reposition.
  // Without this, Floating UI measures the element before the URL text renders,
  // computing position based on the wrong (smaller) dimensions.
  useEffect(() => {
    if (href && editor && !editor.isDestroyed) {
      editor.view.dispatch(editor.state.tr.setMeta('bubbleMenu', 'updatePosition'));
    }
  }, [href, editor]);

  const getReferencedVirtualElement = useCallback(() => {
    if (!editor) return null;
    const { from } = editor.state.selection;
    const range = getLinkRange(editor.state, from);
    if (!range) return null;

    const startCoords = editor.view.coordsAtPos(range.from);
    const endCoords = editor.view.coordsAtPos(range.to);

    return {
      getBoundingClientRect: () =>
        new DOMRect(
          startCoords.left,
          startCoords.top,
          endCoords.left - startCoords.left,
          startCoords.bottom - startCoords.top,
        ),
    };
  }, [editor]);

  if (!editor) return null;

  const truncatedUrl = href && href.length > 40 ? href.slice(0, 40) + '...' : href;

  const handleRemoveLink = () => {
    editor.chain().focus().extendMarkRange('link').unsetLink().run();
  };

  const handleOpenLink = () => {
    if (href) {
      window.open(href, '_blank', 'noopener,noreferrer');
    }
  };

  return (
    <BubbleMenu
      editor={editor}
      pluginKey="linkBubbleMenu"
      shouldShow={shouldShow}
      getReferencedVirtualElement={getReferencedVirtualElement}
      updateDelay={0}
      options={{ placement: 'top', offset: 4 }}
      className="flex items-center gap-1 rounded-lg border border-gray-200 bg-white px-2 py-1 shadow-lg dark:border-gray-700 dark:bg-gray-800"
    >
      <span
        className="max-w-[200px] truncate text-sm text-gray-700 dark:text-gray-300"
        title={href ?? undefined}
      >
        {truncatedUrl}
      </span>
      <div className="mx-1 h-4 w-px bg-gray-200 dark:bg-gray-700" />
      <IconButton onPress={handleOpenLink} aria-label="Open link" size="sm">
        <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" />
      </IconButton>
      <IconButton onPress={onEditLink} aria-label="Edit link" size="sm">
        <PencilIcon className="h-3.5 w-3.5" />
      </IconButton>
      <IconButton onPress={handleRemoveLink} aria-label="Remove link" variant="danger" size="sm">
        <TrashIcon className="h-3.5 w-3.5" />
      </IconButton>
    </BubbleMenu>
  );
}
