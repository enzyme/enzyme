import type { SuggestionOptions, SuggestionProps } from '@tiptap/suggestion';
import { ChannelSuggestionList, type ChannelOption, type ChannelSuggestionRef } from './ChannelSuggestionList';

/**
 * Create suggestion configuration for TipTap channel mention extension.
 * Triggers on # character for #channel typeahead.
 */
export function createChannelSuggestion(
  getChannels: (query: string) => ChannelOption[],
  onOpenChange?: (open: boolean) => void,
): Omit<SuggestionOptions<ChannelOption>, 'editor'> {
  return {
    char: '#',
    allowSpaces: false,

    items: ({ query }) => {
      return getChannels(query);
    },

    render: () => {
      let component: ReturnType<typeof import('react-dom/client').createRoot> | null = null;
      let popup: HTMLDivElement | null = null;
      let ref: ChannelSuggestionRef | null = null;

      return {
        onStart: (props: SuggestionProps<ChannelOption>) => {
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

          import('react-dom/client').then(({ createRoot }) => {
            if (!popup) return;
            component = createRoot(popup);
            component.render(
              <ChannelSuggestionList
                items={props.items}
                command={(item) => {
                  props.command({
                    id: item.id,
                    name: item.name,
                    type: item.type,
                  } as ChannelOption);
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

        onUpdate: (props: SuggestionProps<ChannelOption>) => {
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
            <ChannelSuggestionList
              items={props.items}
              command={(item) => {
                props.command({
                  id: item.id,
                  name: item.name,
                  type: item.type,
                } as ChannelOption);
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
