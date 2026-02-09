import { Node, mergeAttributes } from '@tiptap/core';
import { Plugin, PluginKey } from '@tiptap/pm/state';
import type { Node as PmNode } from '@tiptap/pm/model';
import { EMOJI_NAME, UNICODE_EMOJI_RE } from '../../../lib/emoji';

function isDocEmojiOnly(doc: PmNode): boolean {
  // Must be a single paragraph
  if (doc.childCount !== 1 || doc.firstChild?.type.name !== 'paragraph') return false;
  const para = doc.firstChild!;

  let emojiCount = 0;
  let hasNonEmoji = false;

  para.forEach((child) => {
    if (hasNonEmoji) return;
    if (child.type.name === 'emojiNode') {
      emojiCount++;
    } else if (child.isText) {
      if (child.text && child.text.trim() !== '') {
        hasNonEmoji = true;
      }
    } else {
      hasNonEmoji = true;
    }
  });

  return !hasNonEmoji && emojiCount >= 1 && emojiCount <= 3;
}

export interface EmojiNodeOptions {
  HTMLAttributes: Record<string, unknown>;
}

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    emojiNode: {
      insertEmoji: (attrs: {
        shortcode: string;
        unicode?: string;
        imageUrl?: string;
      }) => ReturnType;
    };
  }
}

export const EmojiNode = Node.create<EmojiNodeOptions>({
  name: 'emojiNode',

  group: 'inline',

  inline: true,

  selectable: true,

  atom: true,

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  addAttributes() {
    return {
      shortcode: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-shortcode'),
        renderHTML: (attributes) => {
          if (!attributes.shortcode) return {};
          return { 'data-shortcode': attributes.shortcode };
        },
      },
      unicode: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-unicode'),
        renderHTML: (attributes) => {
          if (!attributes.unicode) return {};
          return { 'data-unicode': attributes.unicode };
        },
      },
      imageUrl: {
        default: null,
        parseHTML: (element) => element.getAttribute('src') || element.getAttribute('data-image-url'),
        renderHTML: (attributes) => {
          if (!attributes.imageUrl) return {};
          return { 'data-image-url': attributes.imageUrl };
        },
      },
    };
  },

  parseHTML() {
    return [
      { tag: 'span[data-type="emoji"]' },
      { tag: 'img[data-type="emoji"]' },
    ];
  },

  renderHTML({ node, HTMLAttributes }) {
    // Custom emoji — render as <img>
    if (node.attrs.imageUrl) {
      return [
        'img',
        mergeAttributes(
          {
            'data-type': 'emoji',
            src: node.attrs.imageUrl,
            alt: `:${node.attrs.shortcode}:`,
            title: `:${node.attrs.shortcode}:`,
            draggable: 'false',
          },
          this.options.HTMLAttributes,
          HTMLAttributes,
          { class: 'emoji-custom' },
        ),
      ];
    }

    // Standard emoji — render as <span> with unicode character
    return [
      'span',
      mergeAttributes(
        { 'data-type': 'emoji' },
        this.options.HTMLAttributes,
        HTMLAttributes,
        { class: 'emoji-node' },
      ),
      node.attrs.unicode || `:${node.attrs.shortcode}:`,
    ];
  },

  renderText({ node }) {
    return `:${node.attrs.shortcode}:`;
  },

  addCommands() {
    return {
      insertEmoji:
        (attrs) =>
        ({ commands }) => {
          return commands.insertContent({
            type: this.name,
            attrs,
          });
        },
    };
  },

  addProseMirrorPlugins() {
    const emojiNodeType = this.type;
    return [
      new Plugin({
        key: new PluginKey('emojiUnicodeReplace'),
        view(editorView) {
          const update = () => {
            editorView.dom.classList.toggle('emoji-only', isDocEmojiOnly(editorView.state.doc));
          };
          update();
          return { update };
        },
        appendTransaction(_transactions, _oldState, newState) {
          const { tr } = newState;
          const re = new RegExp(UNICODE_EMOJI_RE.source, 'g');
          let modified = false;

          newState.doc.descendants((node, pos) => {
            if (!node.isText || !node.text) return;

            re.lastIndex = 0;
            let match;
            while ((match = re.exec(node.text)) !== null) {
              const shortcode = EMOJI_NAME[match[0]];
              if (!shortcode) continue;

              const from = pos + match.index;
              const to = from + match[0].length;
              const emojiNode = emojiNodeType.create({
                shortcode,
                unicode: match[0],
              });
              tr.replaceWith(
                tr.mapping.map(from),
                tr.mapping.map(to),
                emojiNode,
              );
              modified = true;
            }
          });

          return modified ? tr : null;
        },
      }),
    ];
  },

});
