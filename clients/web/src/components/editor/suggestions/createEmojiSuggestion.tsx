import type { MutableRefObject } from 'react';
import type { SuggestionOptions, SuggestionProps } from '@tiptap/suggestion';
import { EmojiSuggestionList, type EmojiOption, type EmojiSuggestionRef } from './EmojiSuggestionList';
import { searchAllEmojis } from '../../../lib/emoji';
import type { CustomEmoji } from '@feather/api-client';

/**
 * Create suggestion configuration for TipTap emoji extension.
 * Triggers on : character for :emoji: typeahead.
 */
export function createEmojiSuggestion(
  customEmojisRef?: MutableRefObject<CustomEmoji[]>,
): Omit<SuggestionOptions<EmojiOption>, 'editor'> {
  return {
    char: ':',
    allowSpaces: false,

    items: ({ query }) => {
      const customEmojis = customEmojisRef?.current || [];
      const customItems = customEmojis.map((e) => ({ name: e.name, url: e.url }));
      return searchAllEmojis(query, 10, customItems).map((r) => ({
        shortcode: r.shortcode,
        emoji: r.emoji || '',
        isCustom: r.isCustom,
        imageUrl: r.imageUrl,
      }));
    },

    render: () => {
      let component: ReturnType<typeof import('react-dom/client').createRoot> | null = null;
      let popup: HTMLDivElement | null = null;
      let ref: EmojiSuggestionRef | null = null;

      return {
        onStart: (props: SuggestionProps<EmojiOption>) => {
          popup = document.createElement('div');
          popup.style.position = 'fixed';
          popup.style.zIndex = '9999';

          document.body.appendChild(popup);

          // Position above cursor
          const rect = props.clientRect?.();
          if (rect && popup) {
            // Initial positioning - will be refined after render
            popup.style.left = `${rect.left}px`;
            popup.style.bottom = `${window.innerHeight - rect.top + 4}px`;
            popup.style.top = 'auto';
          }

          import('react-dom/client').then(({ createRoot }) => {
            if (!popup) return;
            component = createRoot(popup);
            component.render(
              <EmojiSuggestionList
                items={props.items}
                command={(item) => {
                  props.command(item);
                }}
                ref={(r) => { ref = r; }}
              />
            );

            // Clamp to viewport after render
            requestAnimationFrame(() => {
              if (!popup) return;
              const popupRect = popup.getBoundingClientRect();

              // Clamp horizontal
              if (popupRect.right > window.innerWidth - 8) {
                popup.style.left = `${window.innerWidth - popupRect.width - 8}px`;
              }
              if (popupRect.left < 8) {
                popup.style.left = '8px';
              }

              // If not enough space above, position below
              if (popupRect.top < 8) {
                const cursorRect = props.clientRect?.();
                if (cursorRect) {
                  popup.style.bottom = 'auto';
                  popup.style.top = `${cursorRect.bottom + 4}px`;
                }
              }
            });
          });
        },

        onUpdate: (props: SuggestionProps<EmojiOption>) => {
          if (!popup) return;

          // Update position
          const rect = props.clientRect?.();
          if (rect) {
            popup.style.left = `${rect.left}px`;
            popup.style.bottom = `${window.innerHeight - rect.top + 4}px`;
            popup.style.top = 'auto';
          }

          // Update content
          component?.render(
            <EmojiSuggestionList
              items={props.items}
              command={(item) => {
                props.command(item);
              }}
              ref={(r) => { ref = r; }}
            />
          );

          // Clamp to viewport
          requestAnimationFrame(() => {
            if (!popup) return;
            const popupRect = popup.getBoundingClientRect();

            if (popupRect.right > window.innerWidth - 8) {
              popup.style.left = `${window.innerWidth - popupRect.width - 8}px`;
            }
            if (popupRect.left < 8) {
              popup.style.left = '8px';
            }

            if (popupRect.top < 8) {
              const cursorRect = props.clientRect?.();
              if (cursorRect) {
                popup.style.bottom = 'auto';
                popup.style.top = `${cursorRect.bottom + 4}px`;
              }
            }
          });
        },

        onKeyDown: (props: { event: KeyboardEvent }) => {
          if (props.event.key === 'Escape') {
            popup?.remove();
            popup = null;
            component?.unmount();
            component = null;
            return true;
          }

          return ref?.onKeyDown(props) ?? false;
        },

        onExit: () => {
          popup?.remove();
          popup = null;
          component?.unmount();
          component = null;
          ref = null;
        },
      };
    },
  };
}
