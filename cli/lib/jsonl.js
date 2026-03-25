/**
 * JSONL (JSON Lines) utilities for mint.
 *
 * JSONL is append-only, concurrent-safe, grep-able, streamable.
 * One JSON object per line. Append = fs.appendFileSync(file, JSON.stringify(entry) + '\n').
 * No read-parse-modify-write cycle. Two agents appending simultaneously just add two lines.
 */
import fs from 'fs';

/**
 * Append a single entry to a JSONL file.
 * Atomic for lines under ~4KB on POSIX (always true for our entries).
 * @param {string} filePath
 * @param {object} entry
 */
export function appendJsonl(filePath, entry) {
  fs.appendFileSync(filePath, JSON.stringify(entry) + '\n');
}

/**
 * Read all entries from a JSONL file.
 * Skips blank lines and invalid JSON (graceful).
 * @param {string} filePath
 * @returns {object[]}
 */
export function readJsonl(filePath) {
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    return content
      .split('\n')
      .filter(line => line.trim())
      .map(line => {
        try { return JSON.parse(line); }
        catch { return null; }
      })
      .filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Read entries from a JSONL file matching a filter.
 * @param {string} filePath
 * @param {function} predicate - (entry) => boolean
 * @returns {object[]}
 */
export function queryJsonl(filePath, predicate) {
  return readJsonl(filePath).filter(predicate);
}

/**
 * Get the most recent entry matching a filter.
 * @param {string} filePath
 * @param {function} predicate
 * @returns {object|null}
 */
export function lastJsonl(filePath, predicate) {
  const entries = readJsonl(filePath);
  for (let i = entries.length - 1; i >= 0; i--) {
    if (!predicate || predicate(entries[i])) return entries[i];
  }
  return null;
}

/**
 * Migrate a markdown table file to JSONL.
 * Reads a markdown table with | delimiters, extracts headers and rows,
 * converts to JSONL entries.
 * @param {string} mdPath - path to markdown file
 * @param {string} jsonlPath - path to output JSONL file
 * @returns {{ migrated: number, skipped: number }}
 */
export function migrateMarkdownTableToJsonl(mdPath, jsonlPath) {
  try {
    const content = fs.readFileSync(mdPath, 'utf8');
    const lines = content.split('\n');

    // Find header row (first line with |)
    let headerIdx = -1;
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].trim().startsWith('|') && lines[i].includes('|')) {
        headerIdx = i;
        break;
      }
    }
    if (headerIdx === -1) return { migrated: 0, skipped: 0 };

    const headers = lines[headerIdx]
      .split('|')
      .map(h => h.trim())
      .filter(Boolean)
      .map(h => h.toLowerCase().replace(/\s+/g, '_'));

    // Skip separator row (|---|---|)
    let migrated = 0;
    let skipped = 0;

    for (let i = headerIdx + 2; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line || !line.startsWith('|')) continue;

      const cells = line.split('|').map(c => c.trim()).filter(Boolean);
      if (cells.length < headers.length) { skipped++; continue; }

      const entry = {};
      for (let j = 0; j < headers.length; j++) {
        entry[headers[j]] = cells[j] || '';
      }
      appendJsonl(jsonlPath, entry);
      migrated++;
    }

    return { migrated, skipped };
  } catch {
    return { migrated: 0, skipped: 0 };
  }
}

/**
 * Format JSONL entries as a pretty table for CLI display.
 * @param {object[]} entries
 * @param {string[]} columns - keys to display
 * @param {object} [options]
 * @param {object} [options.widths] - { key: maxWidth }
 * @param {object} [options.labels] - { key: 'Display Label' }
 * @returns {string}
 */
export function formatTable(entries, columns, options = {}) {
  const { widths = {}, labels = {} } = options;

  // Calculate column widths
  const colWidths = {};
  for (const col of columns) {
    const label = labels[col] || col.toUpperCase();
    const maxData = entries.reduce((max, e) => Math.max(max, String(e[col] || '').length), 0);
    colWidths[col] = Math.min(widths[col] || 60, Math.max(label.length, maxData));
  }

  // Header
  const header = columns
    .map(col => (labels[col] || col.toUpperCase()).padEnd(colWidths[col]))
    .join('  ');

  // Rows
  const rows = entries.map(entry =>
    columns
      .map(col => String(entry[col] || '').slice(0, colWidths[col]).padEnd(colWidths[col]))
      .join('  ')
  );

  return `  ${header}\n  ${'─'.repeat(header.length)}\n${rows.map(r => `  ${r}`).join('\n')}`;
}
