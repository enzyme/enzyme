/**
 * Parse mrkdwn text into structured segments for rendering.
 *
 * Supported syntax:
 * - *bold* for bold text
 * - _italic_ for italic text
 * - ~strikethrough~ for strikethrough
 * - `code` for inline code
 * - ```lang\ncode\n``` for code blocks
 * - > for blockquotes (at line start)
 * - • or - for bullet lists
 * - 1. for numbered lists
 * - <@userId> for user mentions
 * - <!here>, <!channel>, <!everyone> for special mentions
 * - <url|text> or <url> for links
 */

export type MrkdwnSegment =
  | { type: 'text'; content: string }
  | { type: 'bold'; content: string }
  | { type: 'italic'; content: string }
  | { type: 'strike'; content: string }
  | { type: 'code'; content: string }
  | { type: 'code_block'; content: string; language?: string }
  | { type: 'blockquote'; segments: MrkdwnSegment[] }
  | { type: 'bullet_list'; items: MrkdwnSegment[][] }
  | { type: 'ordered_list'; items: MrkdwnSegment[][] }
  | { type: 'user_mention'; userId: string }
  | { type: 'special_mention'; mentionType: string }
  | { type: 'channel_mention'; channelId: string }
  | { type: 'link'; url: string; text: string }
  | { type: 'emoji_shortcode'; name: string }
  | { type: 'line_break' };

/**
 * Parse mrkdwn content into segments.
 * Handles both block-level and inline formatting.
 */
export function parseMrkdwn(content: string): MrkdwnSegment[] {
  if (!content) return [];

  const segments: MrkdwnSegment[] = [];
  const lines = content.split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Check for code block
    if (line.startsWith('```')) {
      const lang = line.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // Skip closing ```
      segments.push({
        type: 'code_block',
        content: codeLines.join('\n'),
        language: lang || undefined,
      });
      continue;
    }

    // Check for blockquote
    if (line.startsWith('> ') || line === '>') {
      const quoteLines: string[] = [];
      while (i < lines.length && (lines[i].startsWith('> ') || lines[i] === '>')) {
        quoteLines.push(lines[i].replace(/^> ?/, ''));
        i++;
      }
      segments.push({
        type: 'blockquote',
        segments: parseInline(quoteLines.join('\n')),
      });
      continue;
    }

    // Check for bullet list
    if (line.startsWith('• ') || line.startsWith('- ')) {
      const items: MrkdwnSegment[][] = [];
      while (i < lines.length && (lines[i].startsWith('• ') || lines[i].startsWith('- '))) {
        items.push(parseInline(lines[i].replace(/^[•-] /, '')));
        i++;
      }
      segments.push({ type: 'bullet_list', items });
      continue;
    }

    // Check for ordered list
    if (/^\d+\. /.test(line)) {
      const items: MrkdwnSegment[][] = [];
      while (i < lines.length && /^\d+\. /.test(lines[i])) {
        items.push(parseInline(lines[i].replace(/^\d+\. /, '')));
        i++;
      }
      segments.push({ type: 'ordered_list', items });
      continue;
    }

    // Regular line - parse inline formatting
    if (line.trim() === '') {
      // Empty line
      if (segments.length > 0) {
        segments.push({ type: 'line_break' });
      }
      i++;
      continue;
    }

    // Parse inline content
    const inlineSegments = parseInline(line);
    segments.push(...inlineSegments);

    // Add line break if not last line and next line is not empty
    if (i < lines.length - 1 && lines[i + 1].trim() !== '') {
      segments.push({ type: 'line_break' });
    }
    i++;
  }

  return segments;
}

/**
 * Parse inline formatting (bold, italic, code, mentions, links).
 */
function parseInline(text: string): MrkdwnSegment[] {
  if (!text) return [];

  const segments: MrkdwnSegment[] = [];
  let remaining = text;

  while (remaining.length > 0) {
    // Try to match patterns in order of precedence
    let matched = false;

    // User mention: <@userId>
    const userMentionMatch = remaining.match(/^<@([^>]+)>/);
    if (userMentionMatch) {
      segments.push({ type: 'user_mention', userId: userMentionMatch[1] });
      remaining = remaining.slice(userMentionMatch[0].length);
      matched = true;
      continue;
    }

    // Special mention: <!here>, <!channel>, <!everyone>
    const specialMentionMatch = remaining.match(/^<!([^>]+)>/);
    if (specialMentionMatch) {
      segments.push({ type: 'special_mention', mentionType: specialMentionMatch[1] });
      remaining = remaining.slice(specialMentionMatch[0].length);
      matched = true;
      continue;
    }

    // Channel mention: <#channelId>
    const channelMentionMatch = remaining.match(/^<#([^>]+)>/);
    if (channelMentionMatch) {
      segments.push({ type: 'channel_mention', channelId: channelMentionMatch[1] });
      remaining = remaining.slice(channelMentionMatch[0].length);
      matched = true;
      continue;
    }

    // Link with text: <url|text>
    const linkWithTextMatch = remaining.match(/^<(https?:\/\/[^|>]+)\|([^>]+)>/);
    if (linkWithTextMatch) {
      segments.push({ type: 'link', url: linkWithTextMatch[1], text: linkWithTextMatch[2] });
      remaining = remaining.slice(linkWithTextMatch[0].length);
      matched = true;
      continue;
    }

    // Plain URL in angle brackets: <url>
    const plainLinkMatch = remaining.match(/^<(https?:\/\/[^>]+)>/);
    if (plainLinkMatch) {
      segments.push({ type: 'link', url: plainLinkMatch[1], text: plainLinkMatch[1] });
      remaining = remaining.slice(plainLinkMatch[0].length);
      matched = true;
      continue;
    }

    // Inline code: `code`
    const codeMatch = remaining.match(/^`([^`]+)`/);
    if (codeMatch) {
      segments.push({ type: 'code', content: codeMatch[1] });
      remaining = remaining.slice(codeMatch[0].length);
      matched = true;
      continue;
    }

    // Emoji shortcode: :name:
    const emojiMatch = remaining.match(/^:([a-zA-Z0-9_+-]+):/);
    if (emojiMatch) {
      segments.push({ type: 'emoji_shortcode', name: emojiMatch[1] });
      remaining = remaining.slice(emojiMatch[0].length);
      matched = true;
      continue;
    }

    // Bold: *text* (must not be followed by another *)
    const boldMatch = remaining.match(/^\*([^*]+)\*/);
    if (boldMatch) {
      segments.push({ type: 'bold', content: boldMatch[1] });
      remaining = remaining.slice(boldMatch[0].length);
      matched = true;
      continue;
    }

    // Italic: _text_
    const italicMatch = remaining.match(/^_([^_]+)_/);
    if (italicMatch) {
      segments.push({ type: 'italic', content: italicMatch[1] });
      remaining = remaining.slice(italicMatch[0].length);
      matched = true;
      continue;
    }

    // Strikethrough: ~text~
    const strikeMatch = remaining.match(/^~([^~]+)~/);
    if (strikeMatch) {
      segments.push({ type: 'strike', content: strikeMatch[1] });
      remaining = remaining.slice(strikeMatch[0].length);
      matched = true;
      continue;
    }

    // If no pattern matched, consume plain text until next special character
    if (!matched) {
      const nextSpecial = remaining.search(/[<`*_~:]/);
      if (nextSpecial === -1) {
        // No more special characters, consume rest as text
        appendText(segments, remaining);
        break;
      } else if (nextSpecial === 0) {
        // Special char at start but didn't match a pattern, consume one char
        appendText(segments, remaining[0]);
        remaining = remaining.slice(1);
      } else {
        // Consume plain text up to next special character
        appendText(segments, remaining.slice(0, nextSpecial));
        remaining = remaining.slice(nextSpecial);
      }
    }
  }

  return segments;
}

/**
 * Helper to append text to segments, merging adjacent text segments.
 */
function appendText(segments: MrkdwnSegment[], text: string): void {
  const last = segments[segments.length - 1];
  if (last?.type === 'text') {
    last.content += text;
  } else {
    segments.push({ type: 'text', content: text });
  }
}
