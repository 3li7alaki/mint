import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { generateDocManifest } from '../cli/lib/detect.js';

const TMP = path.join(import.meta.dir, '.tmp-doc-manifest');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('generateDocManifest', () => {
  test('returns valid schema', () => {
    const manifest = generateDocManifest(TMP);
    expect(manifest.$schema).toBe('doc-manifest-v1');
    expect(Array.isArray(manifest.docs)).toBe(true);
  });

  test('detects README.md', () => {
    fs.writeFileSync(path.join(TMP, 'README.md'), '# Test');
    const manifest = generateDocManifest(TMP);
    const readme = manifest.docs.find(d => d.path === 'README.md');
    expect(readme).toBeDefined();
    expect(readme.trigger).toBe('on-task-complete');
    expect(readme.description).toContain('Public-facing');
  });

  test('detects CONTRIBUTING.md', () => {
    fs.writeFileSync(path.join(TMP, 'CONTRIBUTING.md'), '# Contributing');
    const manifest = generateDocManifest(TMP);
    const contrib = manifest.docs.find(d => d.path === 'CONTRIBUTING.md');
    expect(contrib).toBeDefined();
    expect(contrib.trigger).toBe('on-architectural-change');
  });

  test('detects CLAUDE.md', () => {
    fs.writeFileSync(path.join(TMP, 'CLAUDE.md'), '# Claude');
    const manifest = generateDocManifest(TMP);
    const claude = manifest.docs.find(d => d.path === 'CLAUDE.md');
    expect(claude).toBeDefined();
    expect(claude.trigger).toBe('on-architectural-change');
  });

  test('scans docs/ directory', () => {
    fs.mkdirSync(path.join(TMP, 'docs'));
    fs.writeFileSync(path.join(TMP, 'docs', 'architecture.md'), '# Arch');
    fs.writeFileSync(path.join(TMP, 'docs', 'conventions.md'), '# Conv');
    const manifest = generateDocManifest(TMP);
    const arch = manifest.docs.find(d => d.path === 'docs/architecture.md');
    const conv = manifest.docs.find(d => d.path === 'docs/conventions.md');
    expect(arch).toBeDefined();
    expect(conv).toBeDefined();
  });

  test('returns empty docs for empty directory', () => {
    const manifest = generateDocManifest(TMP);
    expect(manifest.docs).toHaveLength(0);
  });

  test('all docs have sections array', () => {
    fs.writeFileSync(path.join(TMP, 'README.md'), '# Test');
    fs.writeFileSync(path.join(TMP, 'CONTRIBUTING.md'), '# Contributing');
    const manifest = generateDocManifest(TMP);
    for (const doc of manifest.docs) {
      expect(Array.isArray(doc.sections)).toBe(true);
    }
  });
});
