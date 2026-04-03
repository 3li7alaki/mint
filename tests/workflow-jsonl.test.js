import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import {
  computeFingerprint,
  fingerprintDistance,
  clusterByFingerprint,
  upsertWorkflowCandidate,
  decayWorkflowCandidates,
  readJsonl,
} from '../cli/lib/jsonl.js';

const TMP = path.join(import.meta.dir, '.tmp-workflow');
const FILE = path.join(TMP, 'candidates.jsonl');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('computeFingerprint', () => {
  test('produces colon-separated type-action string', () => {
    const ops = [
      { type: 'git', action: 'fetch', target: 'origin/main' },
      { type: 'edit', files: ['src/foo.ts'] },
      { type: 'gate', result: 'pass' },
      { type: 'commit', message: 'fix: something' },
    ];
    expect(computeFingerprint(ops)).toBe('git-fetch:edit:gate:commit');
  });

  test('uses just type when no action field', () => {
    const ops = [{ type: 'edit' }, { type: 'gate' }];
    expect(computeFingerprint(ops)).toBe('edit:gate');
  });

  test('returns empty string for empty array', () => {
    expect(computeFingerprint([])).toBe('');
  });

  test('returns empty string for null/undefined', () => {
    expect(computeFingerprint(null)).toBe('');
    expect(computeFingerprint(undefined)).toBe('');
  });

  test('single operation', () => {
    expect(computeFingerprint([{ type: 'commit' }])).toBe('commit');
  });

  test('single operation with action', () => {
    expect(computeFingerprint([{ type: 'git', action: 'push' }])).toBe('git-push');
  });
});

describe('fingerprintDistance', () => {
  test('identical fingerprints return 0', () => {
    expect(fingerprintDistance('git-fetch:edit:gate:commit', 'git-fetch:edit:gate:commit')).toBe(0);
  });

  test('single segment difference returns 1', () => {
    expect(fingerprintDistance('git-fetch:edit:gate:commit', 'git-fetch:edit:commit')).toBe(1);
  });

  test('completely different fingerprints return correct distance', () => {
    expect(fingerprintDistance('git-fetch:edit', 'gate:commit:review')).toBe(3);
  });

  test('empty vs non-empty returns length of non-empty', () => {
    expect(fingerprintDistance('', 'git-fetch:edit:gate')).toBe(3);
    expect(fingerprintDistance('git-fetch:edit', '')).toBe(2);
  });

  test('both empty returns 0', () => {
    expect(fingerprintDistance('', '')).toBe(0);
  });

  test('single substitution returns 1', () => {
    expect(fingerprintDistance('edit:gate:commit', 'edit:lint:commit')).toBe(1);
  });
});

describe('clusterByFingerprint', () => {
  test('groups similar traces together', () => {
    const traces = [
      { fingerprint: 'git-fetch:edit:gate:commit', task: 'a' },
      { fingerprint: 'git-fetch:edit:commit', task: 'b' },        // distance 1 from a
      { fingerprint: 'review:approve:deploy', task: 'c' },         // very different
      { fingerprint: 'review:approve:deploy:notify', task: 'd' },  // distance 1 from c
    ];
    const clusters = clusterByFingerprint(traces, 2);
    expect(clusters).toHaveLength(2);

    // Find which cluster has task a
    const clusterA = clusters.find(c => c.some(t => t.task === 'a'));
    const clusterC = clusters.find(c => c.some(t => t.task === 'c'));
    expect(clusterA).toHaveLength(2);
    expect(clusterC).toHaveLength(2);
  });

  test('each trace in its own cluster when all different', () => {
    const traces = [
      { fingerprint: 'a:b:c:d:e' },
      { fingerprint: 'x:y:z' },
    ];
    const clusters = clusterByFingerprint(traces, 0);
    expect(clusters).toHaveLength(2);
  });

  test('all traces in one cluster when all identical', () => {
    const traces = [
      { fingerprint: 'edit:gate:commit' },
      { fingerprint: 'edit:gate:commit' },
      { fingerprint: 'edit:gate:commit' },
    ];
    const clusters = clusterByFingerprint(traces);
    expect(clusters).toHaveLength(1);
    expect(clusters[0]).toHaveLength(3);
  });

  test('empty input returns empty', () => {
    expect(clusterByFingerprint([])).toEqual([]);
  });
});

describe('upsertWorkflowCandidate', () => {
  test('creates new candidate on first insert', () => {
    const result = upsertWorkflowCandidate(FILE, {
      fingerprint: 'git-fetch:edit:gate:commit',
      proposedName: 'fetch-fix-commit',
    });

    expect(result.action).toBe('created');
    expect(result.confidence).toBe(1);

    const entries = readJsonl(FILE);
    expect(entries).toHaveLength(1);
    expect(entries[0].fingerprint).toBe('git-fetch:edit:gate:commit');
    expect(entries[0].occurrences).toBe(1);
    expect(entries[0].confidence).toBe(1);
    expect(entries[0].proposedName).toBe('fetch-fix-commit');
  });

  test('increments confidence on duplicate fingerprint', () => {
    upsertWorkflowCandidate(FILE, { fingerprint: 'edit:gate:commit' });
    const result = upsertWorkflowCandidate(FILE, { fingerprint: 'edit:gate:commit' });

    expect(result.action).toBe('updated');
    expect(result.confidence).toBe(2);

    const entries = readJsonl(FILE);
    expect(entries).toHaveLength(1);
    expect(entries[0].confidence).toBe(2);
    expect(entries[0].occurrences).toBe(2);
  });

  test('matches similar fingerprints (distance <= 1)', () => {
    upsertWorkflowCandidate(FILE, { fingerprint: 'edit:gate:commit' });
    const result = upsertWorkflowCandidate(FILE, { fingerprint: 'edit:lint:commit' }); // distance 1

    expect(result.action).toBe('updated');
    expect(readJsonl(FILE)).toHaveLength(1);
  });

  test('creates separate entry for distant fingerprints', () => {
    upsertWorkflowCandidate(FILE, { fingerprint: 'edit:gate:commit' });
    upsertWorkflowCandidate(FILE, { fingerprint: 'review:approve:deploy' });

    expect(readJsonl(FILE)).toHaveLength(2);
  });

  test('works when file does not exist yet', () => {
    const newFile = path.join(TMP, 'nonexistent.jsonl');
    const result = upsertWorkflowCandidate(newFile, {
      fingerprint: 'edit:commit',
    });
    expect(result.action).toBe('created');
    expect(readJsonl(newFile)).toHaveLength(1);
  });

  test('merges example traces', () => {
    upsertWorkflowCandidate(FILE, {
      fingerprint: 'edit:commit',
      exampleTraces: ['session-1'],
    });
    upsertWorkflowCandidate(FILE, {
      fingerprint: 'edit:commit',
      exampleTraces: ['session-2'],
    });

    const entries = readJsonl(FILE);
    expect(entries[0].exampleTraces).toEqual(['session-1', 'session-2']);
  });
});

describe('decayWorkflowCandidates', () => {
  test('decays stale candidates', () => {
    const old = new Date(Date.now() - 40 * 24 * 60 * 60 * 1000).toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      fingerprint: 'edit:gate:commit',
      confidence: 3, occurrences: 3, firstSeen: old, lastSeen: old,
    }) + '\n');

    const result = decayWorkflowCandidates(FILE, 30);
    expect(result.decayed).toBe(1);

    const entries = readJsonl(FILE);
    expect(entries[0].confidence).toBe(2);
  });

  test('removes candidates at confidence 0', () => {
    const old = new Date(Date.now() - 40 * 24 * 60 * 60 * 1000).toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      fingerprint: 'edit:commit',
      confidence: 1, occurrences: 1, firstSeen: old, lastSeen: old,
    }) + '\n');

    const result = decayWorkflowCandidates(FILE, 30);
    expect(result.removed).toBe(1);
    expect(readJsonl(FILE)).toHaveLength(0);
  });

  test('leaves fresh candidates alone', () => {
    const recent = new Date().toISOString();
    fs.writeFileSync(FILE, JSON.stringify({
      fingerprint: 'edit:gate:commit',
      confidence: 2, occurrences: 2, firstSeen: recent, lastSeen: recent,
    }) + '\n');

    const result = decayWorkflowCandidates(FILE, 30);
    expect(result.decayed).toBe(0);
    expect(result.removed).toBe(0);
  });

  test('handles empty file', () => {
    const result = decayWorkflowCandidates(FILE, 30);
    expect(result.decayed).toBe(0);
    expect(result.removed).toBe(0);
  });
});
