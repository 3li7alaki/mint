import { describe, test, expect } from 'bun:test';
import { classifyGateTier, gatesForTier, DEFAULT_TIER_PATTERNS } from '../cli/lib/gate-tiers.js';

describe('classifyGateTier', () => {
  // ─── Skip tier ──────────────────────────────────────────────────────────────
  test('markdown files → skip', () => {
    const result = classifyGateTier([{ path: 'README.md' }, { path: 'docs/architecture.md' }]);
    expect(result.tier).toBe('skip');
  });

  test('image files → skip', () => {
    const result = classifyGateTier([{ path: 'assets/logo.png' }, { path: 'icon.svg' }]);
    expect(result.tier).toBe('skip');
  });

  test('.mint files → skip', () => {
    const result = classifyGateTier([{ path: '.mint/config.json' }, { path: '.mint/issues.jsonl' }]);
    expect(result.tier).toBe('skip');
  });

  test('docs directory → skip', () => {
    const result = classifyGateTier([{ path: 'docs/conventions.md' }, { path: 'docs/plans/feature.md' }]);
    expect(result.tier).toBe('skip');
  });

  test('LICENSE → skip', () => {
    const result = classifyGateTier([{ path: 'LICENSE' }]);
    expect(result.tier).toBe('skip');
  });

  // ─── Quick tier ─────────────────────────────────────────────────────────────
  test('new test files → quick', () => {
    const result = classifyGateTier([{ path: 'tests/auth.test.ts', status: 'new' }]);
    expect(result.tier).toBe('quick');
  });

  test('modified test files → full (not quick)', () => {
    const result = classifyGateTier([{ path: 'tests/auth.test.ts', status: 'modified' }]);
    expect(result.tier).toBe('full');
  });

  test('CSS files → quick', () => {
    const result = classifyGateTier([{ path: 'styles/main.css' }]);
    expect(result.tier).toBe('quick');
  });

  test('SCSS modules → quick', () => {
    const result = classifyGateTier([{ path: 'components/Button.module.scss' }]);
    expect(result.tier).toBe('quick');
  });

  // ─── Full tier ──────────────────────────────────────────────────────────────
  test('source files → full', () => {
    const result = classifyGateTier([{ path: 'src/auth/handler.ts' }]);
    expect(result.tier).toBe('full');
  });

  test('CLI files → full', () => {
    const result = classifyGateTier([{ path: 'cli/commands/init.js' }]);
    expect(result.tier).toBe('full');
  });

  test('config files → full', () => {
    const result = classifyGateTier([{ path: 'tsconfig.json' }]);
    expect(result.tier).toBe('full');
  });

  test('package.json → full', () => {
    const result = classifyGateTier([{ path: 'package.json' }]);
    expect(result.tier).toBe('full');
  });

  test('unmatched files default to full', () => {
    const result = classifyGateTier([{ path: 'unknown/weird-file.xyz' }]);
    expect(result.tier).toBe('full');
  });

  // ─── Mixed tiers (highest wins) ────────────────────────────────────────────
  test('mix of skip + full → full', () => {
    const result = classifyGateTier([
      { path: 'README.md' },
      { path: 'src/auth.ts' },
    ]);
    expect(result.tier).toBe('full');
    expect(result.breakdown.skip).toContain('README.md');
    expect(result.breakdown.full).toContain('src/auth.ts');
  });

  test('mix of skip + quick → quick', () => {
    const result = classifyGateTier([
      { path: 'README.md' },
      { path: 'styles/main.css' },
    ]);
    expect(result.tier).toBe('quick');
  });

  test('mix of quick + full → full', () => {
    const result = classifyGateTier([
      { path: 'styles/main.css' },
      { path: 'src/api/routes.ts' },
    ]);
    expect(result.tier).toBe('full');
  });

  test('all skip files → skip', () => {
    const result = classifyGateTier([
      { path: 'README.md' },
      { path: 'docs/guide.md' },
      { path: 'CHANGELOG.md' },
      { path: 'assets/logo.png' },
    ]);
    expect(result.tier).toBe('skip');
  });

  // ─── Breakdown tracking ─────────────────────────────────────────────────────
  test('breakdown correctly categorizes each file', () => {
    const result = classifyGateTier([
      { path: 'README.md' },
      { path: 'styles/app.css' },
      { path: 'src/main.ts' },
    ]);
    expect(result.tier).toBe('full');
    expect(result.breakdown.skip).toEqual(['README.md']);
    expect(result.breakdown.quick).toEqual(['styles/app.css']);
    expect(result.breakdown.full).toEqual(['src/main.ts']);
  });

  // ─── Custom config override ─────────────────────────────────────────────────
  test('custom skip patterns override defaults', () => {
    const result = classifyGateTier(
      [{ path: 'src/generated.ts' }],
      { skip: ['src/generated.*'], quick: [], full: [] }
    );
    expect(result.tier).toBe('skip');
  });

  test('custom full patterns for specific directories', () => {
    const result = classifyGateTier(
      [{ path: 'scripts/deploy.sh' }],
      { skip: [], quick: [], full: ['scripts/**'] }
    );
    expect(result.tier).toBe('full');
  });

  // ─── String input (backward compat) ────────────────────────────────────────
  test('accepts plain string paths', () => {
    const result = classifyGateTier(['README.md', 'docs/guide.md']);
    expect(result.tier).toBe('skip');
  });
});

describe('gatesForTier', () => {
  const config = { lint: 'eslint .', types: 'tsc --noEmit', tests: 'bun test' };

  test('skip tier → no gates', () => {
    const gates = gatesForTier('skip', config);
    expect(gates.lint).toBe(false);
    expect(gates.types).toBe(false);
    expect(gates.tests).toBe(false);
  });

  test('quick tier → types only', () => {
    const gates = gatesForTier('quick', config);
    expect(gates.lint).toBe(false);
    expect(gates.types).toBe('tsc --noEmit');
    expect(gates.tests).toBe(false);
  });

  test('full tier → all gates', () => {
    const gates = gatesForTier('full', config);
    expect(gates.lint).toBe('eslint .');
    expect(gates.types).toBe('tsc --noEmit');
    expect(gates.tests).toBe('bun test');
  });

  test('handles disabled gates in config', () => {
    const gates = gatesForTier('full', { lint: false, types: false, tests: 'bun test' });
    expect(gates.lint).toBe(false);
    expect(gates.types).toBe(false);
    expect(gates.tests).toBe('bun test');
  });
});

describe('DEFAULT_TIER_PATTERNS', () => {
  test('skip patterns include common non-code files', () => {
    expect(DEFAULT_TIER_PATTERNS.skip).toContain('*.md');
    expect(DEFAULT_TIER_PATTERNS.skip).toContain('*.png');
    expect(DEFAULT_TIER_PATTERNS.skip).toContain('docs/**');
  });

  test('quick patterns include new test files', () => {
    expect(DEFAULT_TIER_PATTERNS.quick).toContain('*.test.ts:new');
    expect(DEFAULT_TIER_PATTERNS.quick).toContain('*.css');
  });

  test('full patterns include source directories', () => {
    expect(DEFAULT_TIER_PATTERNS.full).toContain('src/**');
    expect(DEFAULT_TIER_PATTERNS.full).toContain('package.json');
  });
});
