import type { EditorState } from '@tiptap/pm/state';

/**
 * Find the start and end positions of the link mark surrounding `pos`.
 * Returns `null` if `pos` is not inside a link.
 */
export function getLinkRange(state: EditorState, pos: number): { from: number; to: number } | null {
  const $pos = state.doc.resolve(pos);
  const linkType = state.schema.marks.link;
  let result: { from: number; to: number } | null = null;

  state.doc.nodesBetween($pos.start(), $pos.end(), (node, nodePos) => {
    if (node.isText && node.marks.some((m) => m.type === linkType)) {
      if (nodePos <= pos && nodePos + node.nodeSize >= pos) {
        result = { from: nodePos, to: nodePos + node.nodeSize };
      }
    }
  });

  return result;
}
