import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs';
import path from 'path';
import {
  fileExists,
  readJsonSafe,
  detectStack,
  detectPackageManager,
  detectGates,
  detectPlugins,
  getCodeGraphProjectName,
} from '../cli/lib/detect.js';

const TMP = path.join(import.meta.dir, '.tmp-detect');

beforeEach(() => {
  fs.mkdirSync(TMP, { recursive: true });
});

afterEach(() => {
  fs.rmSync(TMP, { recursive: true, force: true });
});

describe('fileExists', () => {
  test('returns true for existing file', () => {
    const f = path.join(TMP, 'exists.txt');
    fs.writeFileSync(f, 'hi');
    expect(fileExists(f)).toBe(true);
  });

  test('returns false for missing file', () => {
    expect(fileExists(path.join(TMP, 'nope.txt'))).toBe(false);
  });
});

describe('readJsonSafe', () => {
  test('parses valid JSON', () => {
    const f = path.join(TMP, 'valid.json');
    fs.writeFileSync(f, '{"a":1}');
    expect(readJsonSafe(f)).toEqual({ a: 1 });
  });

  test('returns null for invalid JSON', () => {
    const f = path.join(TMP, 'bad.json');
    fs.writeFileSync(f, 'not json');
    expect(readJsonSafe(f)).toBeNull();
  });

  test('returns null for missing file', () => {
    expect(readJsonSafe(path.join(TMP, 'missing.json'))).toBeNull();
  });
});

describe('detectStack', () => {
  test('detects nuxt', () => {
    fs.writeFileSync(path.join(TMP, 'nuxt.config.ts'), '');
    expect(detectStack(TMP)).toBe('nuxt');
  });

  test('detects nextjs', () => {
    fs.writeFileSync(path.join(TMP, 'next.config.js'), '');
    expect(detectStack(TMP)).toBe('nextjs');
  });

  test('detects svelte', () => {
    fs.writeFileSync(path.join(TMP, 'svelte.config.js'), '');
    expect(detectStack(TMP)).toBe('svelte');
  });

  test('detects angular', () => {
    fs.writeFileSync(path.join(TMP, 'angular.json'), '{}');
    expect(detectStack(TMP)).toBe('angular');
  });

  test('detects vue via vite + vue dep', () => {
    fs.writeFileSync(path.join(TMP, 'vite.config.ts'), '');
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: { vue: '^3' },
    }));
    expect(detectStack(TMP)).toBe('vue');
  });

  test('detects react via vite + react dep', () => {
    fs.writeFileSync(path.join(TMP, 'vite.config.ts'), '');
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: { react: '^18' },
    }));
    expect(detectStack(TMP)).toBe('react');
  });

  test('detects plain vite without framework deps', () => {
    fs.writeFileSync(path.join(TMP, 'vite.config.ts'), '');
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: {},
    }));
    expect(detectStack(TMP)).toBe('vite');
  });

  test('detects django via manage.py', () => {
    fs.writeFileSync(path.join(TMP, 'manage.py'), '');
    expect(detectStack(TMP)).toBe('django');
  });

  test('detects python via requirements.txt', () => {
    fs.writeFileSync(path.join(TMP, 'requirements.txt'), '');
    expect(detectStack(TMP)).toBe('python');
  });

  test('detects go', () => {
    fs.writeFileSync(path.join(TMP, 'go.mod'), '');
    expect(detectStack(TMP)).toBe('go');
  });

  test('detects rust', () => {
    fs.writeFileSync(path.join(TMP, 'Cargo.toml'), '');
    expect(detectStack(TMP)).toBe('rust');
  });

  test('detects react via package.json', () => {
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: { react: '^18' },
    }));
    expect(detectStack(TMP)).toBe('react');
  });

  test('detects node via express', () => {
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      dependencies: { express: '^4' },
    }));
    expect(detectStack(TMP)).toBe('node');
  });

  test('returns none for empty dir', () => {
    expect(detectStack(TMP)).toBe('none');
  });
});

describe('detectPackageManager', () => {
  test('detects bun', () => {
    fs.writeFileSync(path.join(TMP, 'bun.lockb'), '');
    expect(detectPackageManager(TMP)).toBe('bun');
  });

  test('detects bun via bun.lock', () => {
    fs.writeFileSync(path.join(TMP, 'bun.lock'), '');
    expect(detectPackageManager(TMP)).toBe('bun');
  });

  test('detects pnpm', () => {
    fs.writeFileSync(path.join(TMP, 'pnpm-lock.yaml'), '');
    expect(detectPackageManager(TMP)).toBe('pnpm');
  });

  test('detects yarn', () => {
    fs.writeFileSync(path.join(TMP, 'yarn.lock'), '');
    expect(detectPackageManager(TMP)).toBe('yarn');
  });

  test('detects npm', () => {
    fs.writeFileSync(path.join(TMP, 'package-lock.json'), '');
    expect(detectPackageManager(TMP)).toBe('npm');
  });

  test('returns none for empty dir', () => {
    expect(detectPackageManager(TMP)).toBe('none');
  });
});

describe('detectGates', () => {
  test('detects vitest', () => {
    fs.writeFileSync(path.join(TMP, 'vitest.config.ts'), '');
    const gates = detectGates('node', 'bun', TMP);
    expect(gates.tests).toContain('vitest');
  });

  test('detects jest', () => {
    fs.writeFileSync(path.join(TMP, 'jest.config.js'), '');
    const gates = detectGates('node', 'npm', TMP);
    expect(gates.tests).toContain('jest');
  });

  test('detects tsconfig for types', () => {
    fs.writeFileSync(path.join(TMP, 'tsconfig.json'), '{}');
    const gates = detectGates('node', 'npm', TMP);
    expect(gates.types).toContain('tsc');
  });

  test('uses typecheck script if available', () => {
    fs.writeFileSync(path.join(TMP, 'tsconfig.json'), '{}');
    fs.writeFileSync(path.join(TMP, 'package.json'), JSON.stringify({
      scripts: { typecheck: 'tsc --noEmit' },
    }));
    const gates = detectGates('node', 'npm', TMP);
    expect(gates.types).toContain('typecheck');
  });

  test('detects biome for lint', () => {
    fs.writeFileSync(path.join(TMP, 'biome.json'), '{}');
    const gates = detectGates('node', 'bun', TMP);
    expect(gates.lint).toContain('biome');
  });

  test('go gates', () => {
    const gates = detectGates('go', 'none', TMP);
    expect(gates.tests).toBe('go test ./...');
    expect(gates.lint).toBe('golangci-lint run');
  });

  test('rust gates', () => {
    const gates = detectGates('rust', 'none', TMP);
    expect(gates.tests).toBe('cargo test');
    expect(gates.types).toBe('cargo check');
    expect(gates.lint).toBe('cargo clippy');
  });
});

describe('detectPlugins', () => {
  test('suggests mint-nuxt for nuxt stack', () => {
    expect(detectPlugins('nuxt')).toContain('mint-nuxt');
  });

  test('suggests mint-nuxt for vue stack', () => {
    expect(detectPlugins('vue')).toContain('mint-nuxt');
  });

  test('returns empty for unknown stack', () => {
    expect(detectPlugins('react')).toEqual([]);
  });
});

describe('getCodeGraphProjectName', () => {
  test('generates hyphenated name from path', () => {
    const name = getCodeGraphProjectName('/home/user/projects/my-app');
    // Should produce something like "home-user-projects-my-app"
    expect(name).toContain('my-app');
    expect(name).not.toContain('/');
  });

  test('handles root-relative paths', () => {
    const name = getCodeGraphProjectName('/tmp/test');
    expect(name).toBe('tmp-test');
  });
});
