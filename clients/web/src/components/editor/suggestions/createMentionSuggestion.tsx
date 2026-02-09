import type { SuggestionOptions, SuggestionProps } from '@tiptap/suggestion';
import type { MentionOption } from '../../../lib/mentions';
import { MentionSuggestionList, type MentionSuggestionRef } from './MentionSuggestionList';

/**
 * Create suggestion configuration for TipTap mention extension
 */
export function createMentionSuggestion(
  getItems: (query: string) => MentionOption[],
  options?: {
    onMentionSelect?: (item: MentionOption) => void;
    onOpenChange?: (open: boolean) => void;
  },
): Omit<SuggestionOptions<MentionOption>, 'editor'> {
  const { onMentionSelect, onOpenChange } = options ?? {};
  return {
    char: '@',
    allowSpaces: false,

    items: ({ query }) => {
      return getItems(query);
    },

    render: () => {
      let component: ReturnType<typeof import('react-dom/client').createRoot> | null = null;
      let popup: HTMLDivElement | null = null;
      let ref: MentionSuggestionRef | null = null;

      return {
        onStart: (props: SuggestionProps<MentionOption>) => {
          onOpenChange?.(true);
          popup = document.createElement('div');
          popup.style.position = 'fixed';
          popup.style.zIndex = '9999';

          document.body.appendChild(popup);

          // Position above cursor
          const rect = props.clientRect?.();
          if (rect && popup) {
            popup.style.left = `${rect.left}px`;
            popup.style.bottom = `${window.innerHeight - rect.top + 4}px`;
            popup.style.top = 'auto';
          }

          // Use dynamic import for createRoot to avoid SSR issues
          import('react-dom/client').then(({ createRoot }) => {
            if (!popup) return;
            component = createRoot(popup);
            component.render(
              <MentionSuggestionList
                items={props.items}
                command={(item) => {
                  props.command({
                    id: item.id,
                    label: item.displayName,
                    type: item.type,
                  } as MentionOption & { label: string });
                  onMentionSelect?.(item);
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

        onUpdate: (props: SuggestionProps<MentionOption>) => {
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
            <MentionSuggestionList
              items={props.items}
              command={(item) => {
                props.command({
                  id: item.id,
                  label: item.displayName,
                  type: item.type,
                } as MentionOption & { label: string });
                onMentionSelect?.(item);
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
          onOpenChange?.(false);
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
