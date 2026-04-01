/**
 * Gitignore management — proper entry-by-entry handling.
 *
 * Fixes the bug where template literal \n gets written literally,
 * mashing all entries onto one line.
 */
import fs from 'fs';
import path from 'path';

const MARKER = '# mint local state';

/** Entries that mint needs gitignored in every project. */
export const MINT_IGNORE_ENTRIES = [
  '.mint/tasks/',
  '.mint/research/',
  '.mint/worktrees/',
  '.mint/plugins/',
  '.mint/sessions/',
  '.mint/ssh-cache.json',
  '.mint/.freeze-list.json',
  '.mint/.browser-sessions.json',
  '.mint/.gate-ledger.jsonl',
  '.mint/.dream-lock',
  '.mint/dream-report.md',
  '.mint/issues-archive.jsonl',
  '.mint/wins-archive.jsonl',
];

/**
 * Ensure a list of entries exist in .gitignore.
 * Adds missing entries one per line under the mint marker section.
 * Creates .gitignore if it doesn't exist.
 *
 * @param {string} projectRoot - project root directory
 * @param {string[]} [entries] - entries to ensure (defaults to MINT_IGNORE_ENTRIES)
 * @returns {{ added: string[], alreadyPresent: string[] }}
 */
export function ensureGitignore(projectRoot, entries = MINT_IGNORE_ENTRIES) {
  const gitignorePath = path.join(projectRoot, '.gitignore');

  let content = '';
  try { content = fs.readFileSync(gitignorePath, 'utf8'); } catch { /* no .gitignore yet */ }

  const added = [];
  const alreadyPresent = [];

  for (const entry of entries) {
    if (content.includes(entry)) {
      alreadyPresent.push(entry);
    } else {
      added.push(entry);
    }
  }

  if (added.length === 0) return { added, alreadyPresent };

  // Build the block to append
  const lines = [];

  // Add marker if not present
  if (!content.includes(MARKER)) {
    lines.push('', MARKER);
  }

  for (const entry of added) {
    lines.push(entry);
  }

  const block = lines.join('\n') + '\n';
  fs.appendFileSync(gitignorePath, block);

  return { added, alreadyPresent };
}
