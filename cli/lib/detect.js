import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

export function fileExists(filePath) {
  try { return fs.existsSync(filePath); }
  catch { return false; }
}

// ─── Global config ──────────────────────────────────────────────────────────

export function getGlobalConfigPath() {
  const home = process.env.HOME || process.env.USERPROFILE || '';
  return path.join(home, '.mint', 'config.json');
}

export function loadGlobalConfig() {
  return readJsonSafe(getGlobalConfigPath()) || {};
}

export function saveGlobalConfig(config) {
  const configPath = getGlobalConfigPath();
  const dir = path.dirname(configPath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');
}

// Keys that are user preferences (not project-specific)
const GLOBAL_KEYS = [
  'reviewers', 'autoCommit', 'tdd', 'isolation', 'modelRouting',
  'instincts', 'hooks', 'definitionOfDone',
];

export { GLOBAL_KEYS };

export function mergeConfigs(globalConfig, projectConfig) {
  if (!globalConfig || Object.keys(globalConfig).length === 0) return projectConfig;
  if (!projectConfig) return null;

  const merged = {};

  // Start with global user preferences as base
  for (const key of GLOBAL_KEYS) {
    if (globalConfig[key] !== undefined) {
      merged[key] = deepClone(globalConfig[key]);
    }
  }

  // Project config overrides everything
  for (const [key, value] of Object.entries(projectConfig)) {
    if (value !== undefined) {
      if (typeof value === 'object' && value !== null && !Array.isArray(value)
          && typeof merged[key] === 'object' && merged[key] !== null && !Array.isArray(merged[key])) {
        merged[key] = deepMerge(merged[key], value);
      } else {
        merged[key] = deepClone(value);
      }
    }
  }

  return merged;
}

function deepClone(obj) {
  if (obj === null || typeof obj !== 'object') return obj;
  if (Array.isArray(obj)) return obj.map(deepClone);
  const clone = {};
  for (const [k, v] of Object.entries(obj)) clone[k] = deepClone(v);
  return clone;
}

function deepMerge(base, override) {
  const result = deepClone(base);
  for (const [k, v] of Object.entries(override)) {
    if (typeof v === 'object' && v !== null && !Array.isArray(v)
        && typeof result[k] === 'object' && result[k] !== null && !Array.isArray(result[k])) {
      result[k] = deepMerge(result[k], v);
    } else {
      result[k] = deepClone(v);
    }
  }
  return result;
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
    // Detect bun projects by lockfile
    if (fileExists(path.join(dir, 'bun.lockb')) || fileExists(path.join(dir, 'bun.lock'))) return 'bun';
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

export function installBun() {
  try {
    execSync('curl -fsSL https://bun.sh/install | bash', { stdio: 'pipe', timeout: 120000 });
    return detectTool('bun');
  } catch {
    return false;
  }
}

export function detectContextMode() {
  const home = process.env.HOME || process.env.USERPROFILE || '';

  // Check Claude Code plugin dirs
  try {
    const pluginDir = path.join(home, '.claude', 'plugins');
    if (fs.existsSync(pluginDir)) {
      const dirs = fs.readdirSync(pluginDir);
      for (const d of dirs) {
        const sub = path.join(pluginDir, d, 'context-mode');
        if (fs.existsSync(sub)) return true;
      }
    }
  } catch { /* ignore */ }

  // Check installed_plugins.json
  try {
    const pluginsFile = path.join(home, '.claude', 'installed_plugins.json');
    if (fs.existsSync(pluginsFile)) {
      const content = fs.readFileSync(pluginsFile, 'utf8');
      if (content.includes('context-mode')) return true;
    }
  } catch { /* ignore */ }

  // Check MCP config file directly (avoids slow `claude mcp list` health checks)
  try {
    const mcpConfigPaths = [
      path.join(home, '.claude', 'claude_desktop_config.json'),
      path.join(home, '.claude.json'),
    ];
    for (const mcpPath of mcpConfigPaths) {
      if (fs.existsSync(mcpPath)) {
        const content = fs.readFileSync(mcpPath, 'utf8');
        if (content.includes('context-mode')) return true;
      }
    }
  } catch { /* ignore */ }

  // Check npm global
  try { execSync('which context-mode', { stdio: 'ignore', timeout: 2000 }); return true; }
  catch { /* ignore */ }

  return false;
}

export function installContextMode() {
  execSync('claude mcp add context-mode -- npx -y context-mode', { stdio: 'pipe', timeout: 60000 });
  return detectContextMode();
}

export function detectPlugins(stack) {
  const map = { nuxt: 'mint-nuxt', vue: 'mint-nuxt' };
  return map[stack] ? [map[stack]] : [];
}

// ─── CLAUDE.md management ───────────────────────────────────────────────────

const CLAUDE_MD_VERSION = '1';
const CLAUDE_MD_START = `<!-- mint:start v${CLAUDE_MD_VERSION} -->`;
const CLAUDE_MD_END = '<!-- mint:end -->';
const CLAUDE_MD_SECTION = `${CLAUDE_MD_START}
## MANDATORY: Use mint for ALL Code Changes

**For ANY task that modifies files in this repo, invoke the \`mint\` skill FIRST.**

This is not optional. Before writing, editing, or deleting any code:
1. Invoke \`mint\` with the task description
2. mint auto-routes to the right mode (quick/plan/ship/research/verify)
3. Follow mint's execution flow with gates and reviews

The only exceptions:
- Pure conversation / answering questions
- Reading files to understand context (no modifications)

If you catch yourself thinking "this is just a small fix" or "I'll just edit one file" — STOP. Invoke mint. Small fixes use quick mode. mint decides the workflow, not you.
${CLAUDE_MD_END}`;

export { CLAUDE_MD_VERSION, CLAUDE_MD_START, CLAUDE_MD_END, CLAUDE_MD_SECTION };

export function generateDocManifest(projectRoot) {
  const manifest = { '$schema': 'doc-manifest-v1', docs: [] };

  const commonDocs = [
    { file: 'README.md', desc: 'Public-facing project documentation', trigger: 'on-task-complete' },
    { file: 'CONTRIBUTING.md', desc: 'Contribution guidelines and project structure', trigger: 'on-architectural-change' },
    { file: 'CHANGELOG.md', desc: 'Version history and release notes', trigger: 'on-task-complete' },
    { file: 'CLAUDE.md', desc: 'AI assistant project instructions', trigger: 'on-architectural-change' },
  ];

  for (const { file, desc, trigger } of commonDocs) {
    if (fileExists(path.join(projectRoot, file))) {
      manifest.docs.push({ path: file, description: desc, trigger, sections: [] });
    }
  }

  // Scan docs/ directory
  const docsDir = path.join(projectRoot, 'docs');
  if (fileExists(docsDir)) {
    try {
      const files = fs.readdirSync(docsDir).filter(f => f.endsWith('.md'));
      for (const file of files) {
        manifest.docs.push({
          path: `docs/${file}`,
          description: `Documentation — ${file.replace('.md', '').replace(/-/g, ' ')}`,
          trigger: 'on-architectural-change',
          sections: [],
        });
      }
    } catch { /* ignore */ }
  }

  return manifest;
}

export function ensureClaudeMd(projectRoot) {
  const claudeMdPath = path.join(projectRoot, 'CLAUDE.md');
  let content = '';
  try { content = fs.readFileSync(claudeMdPath, 'utf8'); } catch { /* no CLAUDE.md yet */ }

  if (content.includes('<!-- mint:start')) {
    // Tagged section exists — check version
    const versionMatch = content.match(/<!-- mint:start v(\d+) -->/);
    const existingVersion = versionMatch ? versionMatch[1] : '0';
    if (existingVersion !== CLAUDE_MD_VERSION) {
      const regex = /<!-- mint:start v\d+ -->[\s\S]*?<!-- mint:end -->/;
      content = content.replace(regex, CLAUDE_MD_SECTION);
      fs.writeFileSync(claudeMdPath, content);
      return 'updated';
    }
    return 'current';
  } else if (content.includes('invoke the `mint` skill') || content.includes('Invoke mint') || content.includes('Use mint for ALL Code Changes')) {
    // Has untagged mint section already — don't duplicate
    return 'current';
  } else {
    const separator = content.length > 0 ? '\n\n' : '';
    fs.writeFileSync(claudeMdPath, content + separator + CLAUDE_MD_SECTION + '\n');
    return 'created';
  }
}
