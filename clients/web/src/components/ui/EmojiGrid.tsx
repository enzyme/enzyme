import { useState, useRef, useCallback, useMemo, useEffect, type ChangeEvent } from 'react';
import { tv } from 'tailwind-variants';
import {
  EMOJI_CATEGORIES,
  EMOJI_NAME,
  COMMON_EMOJIS,
  SKIN_TONES,
  SKIN_TONE_EMOJIS,
  searchEmojis,
  applySkinTone,
  getSavedSkinTone,
  saveSkinTone,
  type SkinTone,
} from '../../lib/emoji';

const styles = tv({
  slots: {
    container: [
      'w-72 flex flex-col',
      'bg-white dark:bg-gray-800',
      'border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg',
    ],
    searchInput: [
      'w-full px-2 py-1.5 text-sm',
      'border border-gray-200 dark:border-gray-700 rounded',
      'bg-white dark:bg-gray-900',
      'text-gray-900 dark:text-white',
      'placeholder-gray-400 dark:placeholder-gray-500',
      'focus:outline-none focus:ring-2 focus:ring-primary-500',
    ],
    categoryBar: [
      'flex items-center gap-0.5 py-1 px-1',
      'border-b border-gray-200 dark:border-gray-700',
    ],
    categoryButton: [
      'w-7 h-7 flex items-center justify-center rounded text-sm',
      'hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors',
    ],
    categoryButtonActive: 'bg-gray-100 dark:bg-gray-700',
    scrollArea: 'overflow-y-auto max-h-72 p-1',
    section: '[content-visibility:auto] [contain-intrinsic-size:auto_none] pt-2 last:pb-2',
    sectionHeader: [
      'text-xs font-medium text-gray-500 dark:text-gray-400',
      'uppercase tracking-wide mb-1 px-1',
    ],
    grid: 'grid grid-cols-8 gap-0.5',
    emojiButton: [
      'w-8 h-8 flex items-center justify-center rounded',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'text-lg transition-colors cursor-pointer',
    ],
    searchResults: 'space-y-0.5 p-1 max-h-72 overflow-y-auto',
    searchResultItem: [
      'w-full flex items-center gap-2 px-2 py-1 rounded',
      'hover:bg-gray-100 dark:hover:bg-gray-700',
      'text-left cursor-pointer transition-colors',
    ],
    searchResultItemActive: 'bg-gray-100 dark:bg-gray-700',
    searchResultEmoji: 'text-lg',
    searchResultShortcode: 'text-sm text-gray-600 dark:text-gray-400',
    noResults: 'text-sm text-gray-500 dark:text-gray-400 text-center py-4',
    footer: [
      'flex items-center gap-2 px-2 py-1.5 h-10',
      'border-t border-gray-200 dark:border-gray-700',
    ],
    footerEmoji: 'text-2xl w-8 text-center',
    footerName: 'text-sm font-medium text-gray-700 dark:text-gray-300 truncate',
    skinToneButton: [
      'text-lg w-8 h-8 flex items-center justify-center rounded',
      'hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors',
    ],
    skinTonePicker: [
      'absolute bottom-full right-0 mb-1',
      'flex items-center gap-0.5 p-1',
      'bg-white dark:bg-gray-800',
      'border border-gray-200 dark:border-gray-700',
      'rounded-lg shadow-lg',
    ],
    skinToneOption: [
      'text-lg w-8 h-8 flex items-center justify-center rounded',
      'hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors',
    ],
    skinToneOptionActive: 'ring-2 ring-primary-500 ring-offset-1 dark:ring-offset-gray-800',
  },
});

interface EmojiGridProps {
  onSelect: (emoji: string) => void;
  autoFocus?: boolean;
}

export function EmojiGrid({ onSelect, autoFocus = true }: EmojiGridProps) {
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState('frequent');
  const [activeSearchIndex, setActiveSearchIndex] = useState(-1);
  const [hoveredEmoji, setHoveredEmoji] = useState<{ emoji: string; name: string } | null>(null);
  const [skinTone, setSkinTone] = useState<SkinTone>(getSavedSkinTone);
  const [showSkinTonePicker, setShowSkinTonePicker] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const sectionRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const skinToneRef = useRef<HTMLDivElement>(null);
  const s = styles();

  const searchResults = useMemo(
    () => (search ? searchEmojis(search, 24) : []),
    [search],
  );
  const isSearching = search.length > 0;

  const handleSearchChange = (e: ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    setActiveSearchIndex(-1);
  };

  const scrollToSection = useCallback((sectionId: string) => {
    setActiveCategory(sectionId);
    const el = sectionRefs.current[sectionId];
    if (el && scrollRef.current) {
      scrollRef.current.scrollTo({ top: el.offsetTop - scrollRef.current.offsetTop, behavior: 'smooth' });
    }
  }, []);

  const handleScroll = useCallback(() => {
    const container = scrollRef.current;
    if (!container || isSearching) return;
    const scrollTop = container.scrollTop;
    let current = 'frequent';
    for (const [id, el] of Object.entries(sectionRefs.current)) {
      if (el && el.offsetTop - container.offsetTop <= scrollTop + 8) {
        current = id;
      }
    }
    setActiveCategory(current);
  }, [isSearching]);

  /** Apply the user's skin tone preference if the emoji supports it. */
  const withSkinTone = useCallback(
    (emoji: string) => (SKIN_TONE_EMOJIS.has(emoji) ? applySkinTone(emoji, skinTone) : emoji),
    [skinTone],
  );

  const handleSkinToneChange = (tone: SkinTone) => {
    setSkinTone(tone);
    saveSkinTone(tone);
    setShowSkinTonePicker(false);
  };

  useEffect(() => {
    if (!showSkinTonePicker) return;
    const handleMouseDown = (e: MouseEvent) => {
      if (skinToneRef.current && !skinToneRef.current.contains(e.target as Node)) {
        setShowSkinTonePicker(false);
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, [showSkinTonePicker]);

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!isSearching) return;
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setActiveSearchIndex((i) => Math.min(i + 1, searchResults.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setActiveSearchIndex((i) => Math.max(i - 1, -1));
        break;
      case 'Enter':
        e.preventDefault();
        if (activeSearchIndex >= 0 && activeSearchIndex < searchResults.length) {
          onSelect(withSkinTone(searchResults[activeSearchIndex].emoji));
        } else if (searchResults.length > 0) {
          onSelect(withSkinTone(searchResults[0].emoji));
        }
        break;
    }
  };

  return (
    <div className={s.container()}>
      <div className="p-2 pb-0">
        <input
          type="text"
          value={search}
          onChange={handleSearchChange}
          onKeyDown={handleSearchKeyDown}
          placeholder="Search emoji..."
          className={s.searchInput()}
          autoFocus={autoFocus}
        />
      </div>

      {/* Category tabs - hidden during search */}
      {!isSearching && (
        <div className={s.categoryBar()}>
          <button
            type="button"
            onClick={() => scrollToSection('frequent')}
            className={s.categoryButton({
              className: activeCategory === 'frequent' ? s.categoryButtonActive() : undefined,
            })}
            aria-label="Frequently used"
            title="Frequently used"
          >
            {'🕐'}
          </button>
          {EMOJI_CATEGORIES.map((cat) => (
            <button
              key={cat.id}
              type="button"
              onClick={() => scrollToSection(cat.id)}
              className={s.categoryButton({
                className: activeCategory === cat.id ? s.categoryButtonActive() : undefined,
              })}
              aria-label={cat.label}
              title={cat.label}
            >
              {cat.icon}
            </button>
          ))}
        </div>
      )}

      {isSearching ? (
        searchResults.length > 0 ? (
          <div className={s.searchResults()}>
            {searchResults.map(({ shortcode, emoji }, i) => {
              const displayed = withSkinTone(emoji);
              return (
                <button
                  key={shortcode}
                  type="button"
                  onClick={() => onSelect(displayed)}
                  className={s.searchResultItem({
                    className: i === activeSearchIndex ? s.searchResultItemActive() : undefined,
                  })}
                >
                  <span className={s.searchResultEmoji()}>{displayed}</span>
                  <span className={s.searchResultShortcode()}>:{shortcode}:</span>
                </button>
              );
            })}
          </div>
        ) : (
          <div className={s.noResults()}>No emoji found</div>
        )
      ) : (
        <div ref={scrollRef} className={s.scrollArea()} onScroll={handleScroll}>
          {/* Frequently used */}
          <div
            ref={(el) => { sectionRefs.current['frequent'] = el; }}
            className={s.section()}
          >
            <div className={s.sectionHeader()}>Frequently used</div>
            <div className={s.grid()}>
              {COMMON_EMOJIS.map((emoji) => {
                const displayed = withSkinTone(emoji);
                return (
                  <button
                    key={emoji}
                    type="button"
                    onClick={() => onSelect(displayed)}
                    onMouseEnter={() => setHoveredEmoji({ emoji: displayed, name: EMOJI_NAME[emoji] || emoji })}
                    onMouseLeave={() => setHoveredEmoji(null)}
                    className={s.emojiButton()}
                    aria-label={emoji}
                  >
                    {displayed}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Category sections */}
          {EMOJI_CATEGORIES.map((cat) => (
            <div
              key={cat.id}
              ref={(el) => { sectionRefs.current[cat.id] = el; }}
              className={s.section()}
            >
              <div className={s.sectionHeader()}>{cat.label}</div>
              <div className={s.grid()}>
                {cat.emojis.map((entry) => {
                  const displayed = withSkinTone(entry.emoji);
                  return (
                    <button
                      key={entry.emoji}
                      type="button"
                      onClick={() => onSelect(displayed)}
                      onMouseEnter={() => setHoveredEmoji({ emoji: displayed, name: entry.aliases[0] })}
                      onMouseLeave={() => setHoveredEmoji(null)}
                      className={s.emojiButton()}
                      aria-label={`:${entry.aliases[0]}:`}
                      title={`:${entry.aliases[0]}:`}
                    >
                      {displayed}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Hover preview footer */}
      <div className={s.footer()}>
        {hoveredEmoji ? (
          <>
            <span className={s.footerEmoji()}>{hoveredEmoji.emoji}</span>
            <span className={s.footerName()}>:{hoveredEmoji.name}:</span>
          </>
        ) : null}

        {/* Skin tone picker */}
        <div ref={skinToneRef} className="relative ml-auto">
          {showSkinTonePicker && (
            <div className={s.skinTonePicker()}>
              {SKIN_TONES.map(({ tone, label, swatch }) => (
                <button
                  key={label}
                  type="button"
                  onClick={() => handleSkinToneChange(tone)}
                  className={s.skinToneOption({
                    className: tone === skinTone ? s.skinToneOptionActive() : undefined,
                  })}
                  aria-label={`${label} skin tone`}
                  title={label}
                >
                  {swatch}
                </button>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={() => setShowSkinTonePicker((v) => !v)}
            className={s.skinToneButton()}
            aria-label="Choose skin tone"
            title="Choose skin tone"
          >
            {SKIN_TONES.find((t) => t.tone === skinTone)?.swatch || '✋'}
          </button>
        </div>
      </div>
    </div>
  );
}
