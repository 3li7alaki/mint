import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

export function fileExists(filePath) {
  try { return fs.existsSync(filePath); }
  catch { return false; }
}

function globExists(dir, pattern) {
  try {
    const files = fs.readdirSync(dir);
    const base = pattern.replace('.*', '');
    return files.some(f => f.startsWith(base));
  } catch { return false; }
}

export function readJsonSafe(filePath) {
  try { return JSON.parse(fs.readFileSync(filePath, 'utf8')); }
  catch { return null; }
}

export function detectStack(dir) {
  dir = dir || process.cwd();

  if (globExists(dir, 'nuxt.config.*')) return 'nuxt';
  if (globExists(dir, 'next.config.*')) return 'nextjs';
  if (globExists(dir, 'svelte.config.*')) return 'svelte';
  if (fileExists(path.join(dir, 'angular.json'))) return 'angular';

  if (globExists(dir, 'vite.config.*')) {
    const pkg = readJsonSafe(path.join(dir, 'package.json'));
    if (pkg) {
      const allDeps = { ...pkg.dependencies, ...pkg.devDependencies };
      if (allDeps.vue || allDeps['@vitejs/plugin-vue']) return 'vue';
      if (allDeps.react || allDeps['@vitejs/plugin-react']) return 'react';
    }
    return 'vite';
  }

  if (fileExists(path.join(dir, 'manage.py'))) return 'django';
  if (fileExists(path.join(dir, 'pyproject.toml'))) {
    try {
      const content = fs.readFileSync(path.join(dir, 'pyproject.toml'), 'utf8');
      if (content.includes('django') || content.includes('Django')) return 'django';
      return 'python';
    } catch { /* fall through */ }
  }
  if (fileExists(path.join(dir, 'requirements.txt'))) return 'python';

  if (fileExists(path.join(dir, 'go.mod'))) return 'go';
  if (fileExists(path.join(dir, 'Cargo.toml'))) return 'rust';

  if (fileExists(path.join(dir, 'package.json'))) {
    const pkg = readJsonSafe(path.join(dir, 'package.json'));
    if (pkg) {
      const allDeps = { ...pkg.dependencies, ...pkg.devDependencies };
      if (allDeps.react) return 'react';
      if (allDeps.vue) return 'vue';
      if (allDeps.express || allDeps.fastify || allDeps.koa) return 'node';
    }
    return 'node';
  }

  return 'none';
}

export function detectPackageManager(dir) {
  dir = dir || process.cwd();
  if (fileExists(path.join(dir, 'bun.lockb')) || fileExists(path.join(dir, 'bun.lock'))) return 'bun';
  if (fileExists(path.join(dir, 'pnpm-lock.yaml'))) return 'pnpm';
  if (fileExists(path.join(dir, 'yarn.lock'))) return 'yarn';
  if (fileExists(path.join(dir, 'package-lock.json'))) return 'npm';
  return 'none';
}

export function detectGates(stack, packageManager, dir) {
  dir = dir || process.cwd();
  const run = packageManager !== 'none' ? `${packageManager} run` : 'npx';
  const gates = { lint: false, types: false, tests: false, coverage: false };

  // Lint
  if (fileExists(path.join(dir, 'biome.json')) || fileExists(path.join(dir, 'biome.jsonc'))) {
    gates.lint = `${run} biome check .`;
  } else if (['eslintrc.js', '.eslintrc.json', '.eslintrc.cjs', 'eslint.config.js', 'eslint.config.mjs'].some(f => fileExists(path.join(dir, f.startsWith('.') ? f : '.' + f)) || fileExists(path.join(dir, f)))) {
    gates.lint = `${run} lint`;
  } else if (stack === 'python') {
    if (detectTool('ruff')) gates.lint = 'ruff check .';
  } else if (stack === 'go') gates.lint = 'golangci-lint run';
  else if (stack === 'rust') gates.lint = 'cargo clippy';

  // Types
  if (fileExists(path.join(dir, 'tsconfig.json'))) {
    gates.types = `${run} tsc --noEmit`;
    const pkg = readJsonSafe(path.join(dir, 'package.json'));
    if (pkg?.scripts?.typecheck) gates.types = `${run} typecheck`;
    else if (pkg?.scripts?.['type-check']) gates.types = `${run} type-check`;
  } else if (stack === 'python' && detectTool('mypy')) gates.types = 'mypy .';
  else if (stack === 'rust') gates.types = 'cargo check';

  // Tests
  if (fileExists(path.join(dir, 'vitest.config.ts')) || fileExists(path.join(dir, 'vitest.config.js'))) {
    gates.tests = `${run} vitest run`;
  } else if (fileExists(path.join(dir, 'jest.config.js')) || fileExists(path.join(dir, 'jest.config.ts'))) {
    gates.tests = `${run} jest`;
  } else if (stack === 'python') gates.tests = 'pytest';
  else if (stack === 'go') gates.tests = 'go test ./...';
  else if (stack === 'rust') gates.tests = 'cargo test';
  else {
    const pkg = readJsonSafe(path.join(dir, 'package.json'));
    if (pkg?.scripts?.test && !pkg.scripts.test.includes('no test specified')) {
      gates.tests = `${run} test`;
    }
  }

  return gates;
}

export function detectTool(name) {
  try { execSync(`which ${name}`, { stdio: 'ignore' }); return true; }
  catch { return false; }
}

export function detectPlugins(stack) {
  const map = { nuxt: 'mint-nuxt', vue: 'mint-nuxt' };
  return map[stack] ? [map[stack]] : [];
}
