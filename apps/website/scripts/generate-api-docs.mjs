#!/usr/bin/env node

/**
 * Generates Markdown API reference docs from server/openapi.yaml.
 * One file per OpenAPI tag, output to apps/website/src/docs/api/.
 *
 * Usage: node apps/website/scripts/generate-api-docs.mjs
 */

import { readFileSync, writeFileSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import yaml from 'js-yaml';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, '..', '..', '..');
const SPEC_PATH = join(ROOT, 'server', 'openapi.yaml');
const OUT_DIR = join(ROOT, 'apps', 'website', 'src', 'docs', 'api');

// Tag display names and ordering within the "API Reference" section
const TAG_CONFIG = {
  server: { title: 'Server', order: 51 },
  auth: { title: 'Authentication', order: 52 },
  users: { title: 'Users', order: 53 },
  workspaces: { title: 'Workspaces', order: 54 },
  channels: { title: 'Channels', order: 55 },
  messages: { title: 'Messages', order: 56 },
  files: { title: 'Files', order: 57 },
  emojis: { title: 'Emojis', order: 58 },
  moderation: { title: 'Moderation', order: 59 },
  sse: { title: 'Server-Sent Events', order: 60 },
};

// ── Helpers ──────────────────────────────────────────────────────────

function loadSpec() {
  return yaml.load(readFileSync(SPEC_PATH, 'utf8'));
}

/** Resolve a local $ref like "#/components/schemas/User" */
function resolve(spec, ref) {
  if (!ref) return undefined;
  const parts = ref.replace('#/', '').split('/');
  let node = spec;
  for (const p of parts) node = node?.[p];
  return node;
}

/** Resolve a value that may be a $ref or inline. */
function resolveObj(spec, obj) {
  if (!obj) return undefined;
  if (obj.$ref) return resolve(spec, obj.$ref);
  return obj;
}

/**
 * Build a fully expanded JSON example from a schema.
 * Resolves all $refs, allOf, and includes ALL fields (not just required).
 * Uses a seen set to prevent infinite recursion on circular refs.
 */
function buildExample(spec, schema, seen = new Set()) {
  if (!schema) return {};

  // Handle $ref — resolve and track to avoid cycles
  if (schema.$ref) {
    const refKey = schema.$ref;
    if (seen.has(refKey)) return '...';
    seen = new Set(seen);
    seen.add(refKey);
    const resolved = resolve(spec, refKey);
    if (!resolved) return {};
    return buildExample(spec, resolved, seen);
  }

  // Handle allOf — merge all sub-schemas
  if (schema.allOf) {
    let result = {};
    for (const item of schema.allOf) {
      const sub = buildExample(spec, item, seen);
      if (typeof sub === 'object' && sub !== null && !Array.isArray(sub)) {
        result = { ...result, ...sub };
      }
    }
    return result;
  }

  // Handle oneOf/anyOf — use the first option
  if (schema.oneOf || schema.anyOf) {
    const options = schema.oneOf || schema.anyOf;
    return buildExample(spec, options[0], seen);
  }

  // Handle arrays
  if (schema.type === 'array') {
    if (schema.items) {
      return [buildExample(spec, schema.items, seen)];
    }
    return [];
  }

  // Handle objects with properties
  if (schema.type === 'object' || schema.properties) {
    const obj = {};
    if (schema.properties) {
      for (const [name, prop] of Object.entries(schema.properties)) {
        obj[name] = buildExample(spec, prop, seen);
      }
    }
    return obj;
  }

  // Primitive types
  return examplePrimitive(schema);
}

function examplePrimitive(prop) {
  if (!prop) return '';
  if (prop.example !== undefined) return prop.example;
  if (prop.enum) return prop.enum[0];
  if (prop.default !== undefined) return prop.default;
  switch (prop.type) {
    case 'string':
      if (prop.format === 'email') return 'user@example.com';
      if (prop.format === 'date-time') return '2025-01-15T09:30:00Z';
      if (prop.format === 'binary') return '(binary)';
      return 'string';
    case 'integer':
    case 'number':
      return 0;
    case 'boolean':
      return true;
    case 'array':
      return [];
    default:
      return '';
  }
}

// ── Markdown generation ─────────────────────────────────────────────

function renderEndpoint(spec, method, path, op) {
  const lines = [];
  const summary = op.summary || op.operationId;
  const needsAuth = op.security?.length > 0;

  lines.push(`## ${summary}`);
  lines.push('');
  lines.push(`\`${method.toUpperCase()} ${path}\``);
  lines.push('');

  if (needsAuth) {
    lines.push('Requires authentication.');
    lines.push('');
  }

  if (op.description) {
    lines.push(op.description.trim());
    lines.push('');
  }

  // Path + query parameters
  const params = (op.parameters || []).map((p) => resolveObj(spec, p));
  if (params.length > 0) {
    lines.push('### Parameters');
    lines.push('');
    lines.push('| Name | In | Type | Description |');
    lines.push('|------|----|------|-------------|');
    for (const p of params) {
      const type = p.schema?.type || 'string';
      lines.push(`| \`${p.name}\` | ${p.in} | ${type} | ${p.description || '-'} |`);
    }
    lines.push('');
  }

  // Request body
  if (op.requestBody) {
    const body = resolveObj(spec, op.requestBody);
    const content = body?.content;
    const jsonSchema = content?.['application/json']?.schema;
    const multipartSchema = content?.['multipart/form-data']?.schema;

    if (multipartSchema && !jsonSchema) {
      lines.push('### Request body');
      lines.push('');
      lines.push('Content type: `multipart/form-data`');
      lines.push('');
      // Show fields as a table for multipart since it's not JSON
      const resolved = resolveObj(spec, multipartSchema) || multipartSchema;
      if (resolved.properties) {
        lines.push('| Field | Type | Required | Description |');
        lines.push('|-------|------|----------|-------------|');
        const required = resolved.required || [];
        for (const [name, prop] of Object.entries(resolved.properties)) {
          const p = resolveObj(spec, prop) || prop;
          const type = p.format === 'binary' ? 'file' : p.type || 'string';
          lines.push(
            `| \`${name}\` | ${type} | ${required.includes(name) ? 'yes' : 'no'} | ${p.description || '-'} |`,
          );
        }
        lines.push('');
      }
    } else if (jsonSchema) {
      const example = buildExample(spec, jsonSchema);
      lines.push('### Request body');
      lines.push('');
      lines.push('```json');
      lines.push(JSON.stringify(example, null, 2));
      lines.push('```');
      lines.push('');
    }
  }

  // Responses — each as a JSON example
  const responses = op.responses;
  if (responses) {
    lines.push('### Responses');
    lines.push('');

    for (const [status, resp] of Object.entries(responses)) {
      const resolved = resolveObj(spec, resp);
      const desc = resolved?.description || '';

      // Check for JSON content
      const jsonContent = resolved?.content?.['application/json'];
      // Check for binary/stream content
      const binaryContent = resolved?.content?.['application/octet-stream'];
      const sseContent = resolved?.content?.['text/event-stream'];

      if (jsonContent?.schema) {
        lines.push(`**\`${status}\` ${desc}**`);
        lines.push('');
        // Prefer response-level example over schema-generated one
        const example = jsonContent.example || buildExample(spec, jsonContent.schema);
        lines.push('```json');
        lines.push(JSON.stringify(example, null, 2));
        lines.push('```');
        lines.push('');
      } else if (binaryContent) {
        lines.push(`- \`${status}\` ${desc} — binary file content`);
      } else if (sseContent) {
        lines.push(`- \`${status}\` ${desc} — \`text/event-stream\``);
      } else if (!jsonContent) {
        lines.push(`- \`${status}\` ${desc}`);
      }
    }
    lines.push('');
  }

  return lines.join('\n');
}

// ── Main ────────────────────────────────────────────────────────────

function main() {
  const spec = loadSpec();
  mkdirSync(OUT_DIR, { recursive: true });

  // Group endpoints by tag
  const byTag = {};
  for (const [path, methods] of Object.entries(spec.paths)) {
    for (const [method, op] of Object.entries(methods)) {
      if (['get', 'post', 'put', 'patch', 'delete'].indexOf(method) === -1) continue;
      const tag = op.tags?.[0] || 'other';
      if (!byTag[tag]) byTag[tag] = [];
      byTag[tag].push({ path, method, op });
    }
  }

  // Find tag descriptions from spec
  const tagDescriptions = {};
  for (const t of spec.tags || []) {
    tagDescriptions[t.name] = t.description;
  }

  let count = 0;

  for (const [tag, endpoints] of Object.entries(byTag)) {
    const config = TAG_CONFIG[tag] || { title: tag, order: 60 + count };
    const title = config.title;

    const lines = [];

    // Front matter
    lines.push('---');
    lines.push(`title: '${title}'`);
    lines.push(`description: '${tagDescriptions[tag] || `${title} API endpoints`}'`);
    lines.push(`section: 'API Reference'`);
    lines.push(`order: ${config.order}`);
    lines.push('---');
    lines.push('');

    // Intro
    if (tagDescriptions[tag]) {
      lines.push(tagDescriptions[tag]);
      lines.push('');
    }

    // Endpoints
    for (const { path, method, op } of endpoints) {
      lines.push(renderEndpoint(spec, method, path, op));
    }

    const filename = `${tag}.md`;
    writeFileSync(join(OUT_DIR, filename), lines.join('\n'));
    console.log(`  ${filename} (${endpoints.length} endpoints)`);
    count++;
  }

  // Write directory data file for Eleventy
  writeFileSync(
    join(OUT_DIR, 'api.json'),
    JSON.stringify(
      {
        layout: 'layouts/doc.njk',
        tags: 'docs',
        permalink: '/docs/api/{{ page.fileSlug }}/',
      },
      null,
      2,
    ) + '\n',
  );

  console.log(`\nGenerated ${count} API reference pages in apps/website/src/docs/api/`);
}

main();
