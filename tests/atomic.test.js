import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import { atomicWriteSync, atomicWriteJsonSync } from '../cli/lib/atomic.js';

const TMP = path.join(import.meta.dir, '.tmp-atomic');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('atomicWriteSync', () => {
  test('writes content to file', () => {
    const file = path.join(TMP, 'test.txt');
    atomicWriteSync(file, 'hello world');
    expect(fs.readFileSync(file, 'utf8')).toBe('hello world');
  });

  test('overwrites existing file', () => {
    const file = path.join(TMP, 'test.txt');
    atomicWriteSync(file, 'first');
    atomicWriteSync(file, 'second');
    expect(fs.readFileSync(file, 'utf8')).toBe('second');
  });

  test('creates parent directories', () => {
    const file = path.join(TMP, 'nested', 'dir', 'test.txt');
    atomicWriteSync(file, 'deep');
    expect(fs.readFileSync(file, 'utf8')).toBe('deep');
  });

  test('no temp files left behind', () => {
    const file = path.join(TMP, 'clean.txt');
    atomicWriteSync(file, 'data');
    const files = fs.readdirSync(TMP);
    expect(files).toEqual(['clean.txt']);
  });
});

describe('atomicWriteJsonSync', () => {
  test('writes pretty JSON with trailing newline', () => {
    const file = path.join(TMP, 'data.json');
    atomicWriteJsonSync(file, { key: 'value', nested: { a: 1 } });
    const content = fs.readFileSync(file, 'utf8');
    expect(content).toBe(JSON.stringify({ key: 'value', nested: { a: 1 } }, null, 2) + '\n');
  });

  test('writes valid JSON that can be parsed back', () => {
    const file = path.join(TMP, 'roundtrip.json');
    const data = { mintInvoked: true, task: 'test', mode: 'quick' };
    atomicWriteJsonSync(file, data);
    const parsed = JSON.parse(fs.readFileSync(file, 'utf8'));
    expect(parsed).toEqual(data);
  });
});
