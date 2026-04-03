import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import {
  evaluateTrustLevel,
  trackSkillUsage,
  promoteTrust,
  detectVariation,
  readJsonl,
  appendJsonl,
} from './jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-trust');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('evaluateTrustLevel', () => {
  test('returns 0 for no history', () => {
    expect(evaluateTrustLevel([])).toBe(0);
    expect(evaluateTrustLevel(null)).toBe(0);
    expect(evaluateTrustLevel(undefined)).toBe(0);
  });

  test('returns 0 for history without generated action', () => {
    const history = [
      { action: 'invoked', success: true, date: '2026-01-01T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(0);
  });

  test('returns 1 for generated but unused', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(1);
  });

  test('returns 1 for generated with fewer than 3 successes', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(1);
  });

  test('returns 2 after 3+ successes', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-04T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(2);
  });

  test('returns 2 after 4 successes (not yet 5)', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-04T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-05T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(2);
  });

  test('returns 3 after 5+ successes, 0 failures', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-04T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-05T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-06T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(3);
  });

  test('demotes on failure — resets to level 1', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-04T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-05T00:00:00Z' },
      { action: 'invoked', success: true, date: '2026-01-06T00:00:00Z' },
      { action: 'invoked', success: false, date: '2026-01-07T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(1);
  });

  test('returns 1 for history with only failures', () => {
    const history = [
      { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' },
      { action: 'invoked', success: false, date: '2026-01-02T00:00:00Z' },
      { action: 'invoked', success: false, date: '2026-01-03T00:00:00Z' },
    ];
    expect(evaluateTrustLevel(history)).toBe(1);
  });
});

describe('detectVariation', () => {
  test('identical fingerprints return match', () => {
    expect(detectVariation('edit:gate:commit', 'edit:gate:commit')).toBe('match');
  });

  test('1 segment diff returns minor', () => {
    expect(detectVariation('edit:gate:commit', 'edit:lint:commit')).toBe('minor');
  });

  test('2 segment diff returns minor', () => {
    expect(detectVariation('edit:gate:commit', 'edit:lint:review')).toBe('minor');
  });

  test('3+ segment diff returns major', () => {
    expect(detectVariation('edit:gate:commit', 'review:approve:deploy')).toBe('major');
  });

  test('respects custom maxDistance', () => {
    // distance is 1, maxDistance is 0 -> major
    expect(detectVariation('edit:gate:commit', 'edit:lint:commit', 0)).toBe('major');
  });

  test('empty vs non-empty returns major', () => {
    expect(detectVariation('', 'edit:gate:commit')).toBe('major');
  });

  test('both empty returns match', () => {
    expect(detectVariation('', '')).toBe('match');
  });
});

describe('trackSkillUsage', () => {
  test('creates history file if missing', () => {
    const skillsDir = path.join(TMP, 'skills');
    trackSkillUsage(skillsDir, 'my-skill', true);

    const historyPath = path.join(skillsDir, 'my-skill', 'history.jsonl');
    expect(fs.existsSync(historyPath)).toBe(true);

    const entries = readJsonl(historyPath);
    expect(entries).toHaveLength(1);
    expect(entries[0].action).toBe('invoked');
    expect(entries[0].success).toBe(true);
  });

  test('appends usage entry to existing history', () => {
    const skillsDir = path.join(TMP, 'skills');
    const skillDir = path.join(skillsDir, 'my-skill');
    fs.mkdirSync(skillDir, { recursive: true });

    const historyPath = path.join(skillDir, 'history.jsonl');
    appendJsonl(historyPath, { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' });

    trackSkillUsage(skillsDir, 'my-skill', true);

    const entries = readJsonl(historyPath);
    expect(entries).toHaveLength(2);
    expect(entries[0].action).toBe('generated');
    expect(entries[1].action).toBe('invoked');
  });

  test('returns current trust level', () => {
    const skillsDir = path.join(TMP, 'skills');
    const skillDir = path.join(skillsDir, 'my-skill');
    fs.mkdirSync(skillDir, { recursive: true });

    const historyPath = path.join(skillDir, 'history.jsonl');
    // Seed with generated + 2 successes
    appendJsonl(historyPath, { action: 'generated', success: true, date: '2026-01-01T00:00:00Z' });
    appendJsonl(historyPath, { action: 'invoked', success: true, date: '2026-01-02T00:00:00Z' });
    appendJsonl(historyPath, { action: 'invoked', success: true, date: '2026-01-03T00:00:00Z' });

    // 3rd success should push to level 2
    const level = trackSkillUsage(skillsDir, 'my-skill', true);
    expect(level).toBe(2);
  });

  test('returns level 0 when no generated entry exists', () => {
    const skillsDir = path.join(TMP, 'skills');
    // No generated entry, just an invocation
    const level = trackSkillUsage(skillsDir, 'new-skill', true);
    expect(level).toBe(0);
  });
});

describe('promoteTrust', () => {
  test('level 2 removes disable-model-invocation', () => {
    const skillDir = path.join(TMP, 'promote-skill');
    fs.mkdirSync(skillDir, { recursive: true });

    const skillPath = path.join(skillDir, 'SKILL.md');
    fs.writeFileSync(skillPath, [
      '---',
      'name: test-skill',
      'disable-model-invocation: true',
      '---',
      '',
      '# Test Skill',
      '',
      'Does stuff.',
    ].join('\n'));

    promoteTrust(skillDir, 2);

    const content = fs.readFileSync(skillPath, 'utf8');
    expect(content).not.toContain('disable-model-invocation');
    expect(content).toContain('name: test-skill');
    expect(content).toContain('# Test Skill');
  });

  test('level 3 adds auto-invoke and removes disable-model-invocation', () => {
    const skillDir = path.join(TMP, 'promote-skill-3');
    fs.mkdirSync(skillDir, { recursive: true });

    const skillPath = path.join(skillDir, 'SKILL.md');
    fs.writeFileSync(skillPath, [
      '---',
      'name: test-skill',
      'disable-model-invocation: true',
      '---',
      '',
      '# Test Skill',
    ].join('\n'));

    promoteTrust(skillDir, 3);

    const content = fs.readFileSync(skillPath, 'utf8');
    expect(content).not.toContain('disable-model-invocation');
    expect(content).toContain('auto-invoke: true');
    expect(content).toContain('name: test-skill');
  });

  test('does not duplicate auto-invoke if already present', () => {
    const skillDir = path.join(TMP, 'promote-idempotent');
    fs.mkdirSync(skillDir, { recursive: true });

    const skillPath = path.join(skillDir, 'SKILL.md');
    fs.writeFileSync(skillPath, [
      '---',
      'name: test-skill',
      'auto-invoke: true',
      '---',
      '',
      '# Test Skill',
    ].join('\n'));

    promoteTrust(skillDir, 3);

    const content = fs.readFileSync(skillPath, 'utf8');
    const matches = content.match(/auto-invoke: true/g);
    expect(matches).toHaveLength(1);
  });

  test('no-op if SKILL.md has no frontmatter', () => {
    const skillDir = path.join(TMP, 'no-frontmatter');
    fs.mkdirSync(skillDir, { recursive: true });

    const skillPath = path.join(skillDir, 'SKILL.md');
    fs.writeFileSync(skillPath, '# Just a heading\n\nNo frontmatter here.\n');

    promoteTrust(skillDir, 2);

    const content = fs.readFileSync(skillPath, 'utf8');
    expect(content).toBe('# Just a heading\n\nNo frontmatter here.\n');
  });
});
