import type { JSONContent } from '@tiptap/react';
import { resolveStandardShortcode } from '@enzyme/shared';

/**
 * Convert mrkdwn string to TipTap JSON content.
 *
 * This is used when editing an existing message - we need to restore
 * the editor state from the stored mrkdwn format.
 */
export function fromMrkdwn(mrkdwn: string): JSONContent {
  if (!mrkdwn) {
    return {
      type: 'doc',
      content: [{ type: 'paragraph' }],
    };
  }

  const lines = mrkdwn.split('\n');
  const content: JSONContent[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Check for code block
    if (line.startsWith('```')) {
      const lang = line.slice(3);
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // Skip closing ```
      content.push({
        type: 'codeBlock',
        attrs: { language: lang || null },
        content: [{ type: 'text', text: codeLines.join('\n') }],
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
      content.push({
        type: 'blockquote',
        content: [
          {
            type: 'paragraph',
            content: parseInline(quoteLines.join('\n')),
          },
        ],
      });
      continue;
    }

    // Check for bullet list
    if (line.startsWith('• ') || line.startsWith('- ')) {
      const items: string[] = [];
      while (i < lines.length && (lines[i].startsWith('• ') || lines[i].startsWith('- '))) {
        items.push(lines[i].replace(/^[•-] /, ''));
        i++;
      }
      content.push({
        type: 'bulletList',
        content: items.map((item) => ({
          type: 'listItem',
          content: [
            {
              type: 'paragraph',
              content: parseInline(item),
            },
          ],
        })),
      });
      continue;
    }

    // Check for ordered list
    const orderedMatch = line.match(/^\d+\. /);
    if (orderedMatch) {
      const items: string[] = [];
      while (i < lines.length && /^\d+\. /.test(lines[i])) {
        items.push(lines[i].replace(/^\d+\. /, ''));
        i++;
      }
      content.push({
        type: 'orderedList',
        content: items.map((item) => ({
          type: 'listItem',
          content: [
            {
              type: 'paragraph',
              content: parseInline(item),
            },
          ],
        })),
      });
      continue;
    }

    // Check for horizontal rule
    if (line === '---') {
      content.push({ type: 'horizontalRule' });
      i++;
      continue;
    }

    // Regular paragraph
    if (line.trim() === '') {
      // Empty line - skip or add empty paragraph
      i++;
      continue;
    }

    content.push({
      type: 'paragraph',
      content: parseInline(line),
    });
    i++;
  }

  // Ensure we have at least one paragraph
  if (content.length === 0) {
    content.push({ type: 'paragraph' });
  }

  return {
    type: 'doc',
    content,
  };
}

/**
 * Parse inline content including mentions, links, and formatting.
 */
function parseInline(text: string): JSONContent[] {
  if (!text) return [];

  const nodes: JSONContent[] = [];
  let remaining = text;

  while (remaining.length > 0) {
    // Try to match patterns
    const patterns = [
      // User mention: <@userId>
      {
        regex: /^<@([^>]+)>/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'userMention',
          attrs: { id: match[1] },
        }),
      },
      // Special mention: <!here>, <!channel>, <!everyone>
      {
        regex: /^<!([^>]+)>/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'specialMention',
          attrs: { id: match[1] },
        }),
      },
      // Channel mention: <#channelId>
      {
        regex: /^<#([^>]+)>/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'channelMention',
          attrs: { id: match[1] },
        }),
      },
      // Link: <url|text>
      {
        regex: /^<(https?:\/\/[^|>]+)\|([^>]+)>/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[2],
          marks: [{ type: 'link', attrs: { href: match[1] } }],
        }),
      },
      // Plain URL in angle brackets: <url>
      {
        regex: /^<(https?:\/\/[^>]+)>/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'link', attrs: { href: match[1] } }],
        }),
      },
      // Emoji shortcode: :shortcode:
      {
        regex: /^:([a-zA-Z0-9_+-]+):/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'emojiNode',
          attrs: {
            shortcode: match[1],
            unicode: resolveStandardShortcode(match[1]) || null,
          },
        }),
      },
      // Code: `text`
      {
        regex: /^`([^`]+)`/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'code' }],
        }),
      },
      // Bold: *text*
      {
        regex: /^\*([^*]+)\*/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'bold' }],
        }),
      },
      // Italic: _text_
      {
        regex: /^_([^_]+)_/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'italic' }],
        }),
      },
      // Underline: ++text++
      {
        regex: /^\+\+([^+]+)\+\+/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'underline' }],
        }),
      },
      // Strikethrough: ~text~
      {
        regex: /^~([^~]+)~/,
        handler: (match: RegExpMatchArray): JSONContent => ({
          type: 'text',
          text: match[1],
          marks: [{ type: 'strike' }],
        }),
      },
    ];

    let matched = false;
    for (const { regex, handler } of patterns) {
      const match = remaining.match(regex);
      if (match) {
        nodes.push(handler(match));
        remaining = remaining.slice(match[0].length);
        matched = true;
        break;
      }
    }

    // If no pattern matched, consume one character as plain text
    if (!matched) {
      // Find the next potential pattern start
      const nextSpecial = remaining.search(/[:<`*_+~]/);
      if (nextSpecial === -1 || nextSpecial === 0) {
        // Consume one character
        const char = remaining[0];
        // Append to previous text node if possible
        const lastNode = nodes[nodes.length - 1];
        if (lastNode?.type === 'text' && !lastNode.marks?.length) {
          lastNode.text = (lastNode.text || '') + char;
        } else {
          nodes.push({ type: 'text', text: char });
        }
        remaining = remaining.slice(1);
      } else {
        // Consume plain text up to next special character
        const plainText = remaining.slice(0, nextSpecial);
        const lastNode = nodes[nodes.length - 1];
        if (lastNode?.type === 'text' && !lastNode.marks?.length) {
          lastNode.text = (lastNode.text || '') + plainText;
        } else {
          nodes.push({ type: 'text', text: plainText });
        }
        remaining = remaining.slice(nextSpecial);
      }
    }
  }

  return nodes;
}
