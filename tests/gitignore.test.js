import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { ensureGitignore, MINT_IGNORE_ENTRIES } from '../cli/lib/gitignore.js';

const TMP = path.join(import.meta.dir, '.tmp-gitignore');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('ensureGitignore', () => {
  test('creates .gitignore with all entries on separate lines', () => {
    ensureGitignore(TMP);
    const content = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    const lines = content.split('\n');

    // Each entry should be its own line — no literal \n
    for (const entry of MINT_IGNORE_ENTRIES) {
      expect(lines).toContain(entry);
    }

    // No line should contain multiple entries mashed together
    for (const line of lines) {
      const matches = MINT_IGNORE_ENTRIES.filter(e => line.includes(e));
      expect(matches.length).toBeLessThanOrEqual(1);
    }
  });

  test('includes marker comment', () => {
    ensureGitignore(TMP);
    const content = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    expect(content).toContain('# mint local state');
  });

  test('appends to existing .gitignore without duplicating', () => {
    fs.writeFileSync(path.join(TMP, '.gitignore'), 'node_modules/\n');
    ensureGitignore(TMP);
    const content = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');

    expect(content).toContain('node_modules/');
    expect(content).toContain('.mint/tasks/');
  });

  test('does not duplicate entries on re-run', () => {
    ensureGitignore(TMP);
    ensureGitignore(TMP);
    const content = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    const taskMatches = content.match(/\.mint\/tasks\//g);
    expect(taskMatches).toHaveLength(1);
  });

  test('adds only missing entries', () => {
    fs.writeFileSync(path.join(TMP, '.gitignore'), '# mint local state\n.mint/tasks/\n.mint/sessions/\n');
    const { added, alreadyPresent } = ensureGitignore(TMP);

    expect(alreadyPresent).toContain('.mint/tasks/');
    expect(alreadyPresent).toContain('.mint/sessions/');
    expect(added).not.toContain('.mint/tasks/');
    expect(added).not.toContain('.mint/sessions/');
    expect(added.length).toBeGreaterThan(0);
  });

  test('supports custom entries', () => {
    ensureGitignore(TMP, ['.env', '.env.local']);
    const content = fs.readFileSync(path.join(TMP, '.gitignore'), 'utf8');
    expect(content).toContain('.env');
    expect(content).toContain('.env.local');
  });

  test('returns correct added/alreadyPresent counts', () => {
    const { added, alreadyPresent } = ensureGitignore(TMP);
    expect(added).toEqual(MINT_IGNORE_ENTRIES);
    expect(alreadyPresent).toEqual([]);

    const second = ensureGitignore(TMP);
    expect(second.added).toEqual([]);
    expect(second.alreadyPresent).toEqual(MINT_IGNORE_ENTRIES);
  });
});
