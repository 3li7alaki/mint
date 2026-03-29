/**
 * Migration runner — applies versioned migrations to a project's .mint/ config.
 *
 * Each migration file in this directory exports:
 *   { from, to, description, steps: [{ name, fn }] }
 *
 * The runner reads mintVersion from config, finds pending migrations, applies them in order.
 */
import fs from 'fs';
import path from 'path';
import { readJsonSafe } from '../lib/detect.js';

/**
 * Compare semver strings. Returns -1, 0, or 1.
 */
function compareSemver(a, b) {
  const pa = (a || '0.0.0').split('.').map(Number);
  const pb = (b || '0.0.0').split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) < (pb[i] || 0)) return -1;
    if ((pa[i] || 0) > (pb[i] || 0)) return 1;
  }
  return 0;
}

/**
 * Load all migration modules from the migrations directory.
 * @returns {Array<{ from, to, description, steps, fileName }>}
 */
export async function loadMigrations() {
  const migrationsDir = path.dirname(new URL(import.meta.url).pathname);
  const files = fs.readdirSync(migrationsDir)
    .filter(f => /^\d+\.\d+\.\d+-to-\d+\.\d+\.\d+\.js$/.test(f))
    .sort();

  const migrations = [];
  for (const file of files) {
    const mod = await import(path.join(migrationsDir, file));
    migrations.push({ ...mod.default, fileName: file });
  }
  return migrations;
}

/**
 * Get pending migrations for a project.
 * @param {string} currentVersion - project's mintVersion
 * @param {string} targetVersion - version to migrate to
 * @param {Array} migrations - loaded migrations
 * @returns {Array} migrations to apply, in order
 */
export function getPendingMigrations(currentVersion, targetVersion, migrations) {
  return migrations.filter(m =>
    compareSemver(m.from, currentVersion) >= 0 &&
    compareSemver(m.to, targetVersion) <= 0
  );
}

/**
 * Run migrations on a project.
 * @param {string} projectRoot
 * @param {string} targetVersion
 * @param {object} [options]
 * @param {function} [options.onStep] - callback(stepName, status) for progress
 * @returns {{ success: boolean, applied: string[], failed?: { migration: string, step: string, error: string } }}
 */
export async function runMigrations(projectRoot, targetVersion, options = {}) {
  const { onStep } = options;
  const configPath = path.join(projectRoot, '.mint', 'config.json');
  const config = readJsonSafe(configPath);

  if (!config) {
    return { success: false, applied: [], failed: { migration: 'pre-check', step: 'config', error: 'No .mint/config.json found' } };
  }

  const currentVersion = config.mintVersion || '0.0.0';
  const migrations = await loadMigrations();
  const pending = getPendingMigrations(currentVersion, targetVersion, migrations);

  if (pending.length === 0) {
    return { success: true, applied: [] };
  }

  // Backup config before any migration
  const backupPath = path.join(projectRoot, '.mint', `.config-backup-${currentVersion}.json`);
  try {
    fs.writeFileSync(backupPath, JSON.stringify(config, null, 2) + '\n');
  } catch { /* backup is best-effort */ }

  const applied = [];

  for (const migration of pending) {
    for (const step of migration.steps) {
      if (onStep) onStep(step.name, 'running');
      try {
        await step.fn(projectRoot, config);
        if (onStep) onStep(step.name, 'done');
      } catch (err) {
        if (onStep) onStep(step.name, 'failed');
        return {
          success: false,
          applied,
          failed: {
            migration: `${migration.from}-to-${migration.to}`,
            step: step.name,
            error: err.message,
          },
        };
      }
    }

    // Update mintVersion after each successful migration
    config.mintVersion = migration.to;
    config.lastUpdated = new Date().toISOString();
    if (!config.appliedMigrations) config.appliedMigrations = [];
    config.appliedMigrations.push(`${migration.from}-to-${migration.to}`);
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');

    applied.push(`${migration.from} → ${migration.to}: ${migration.description}`);
  }

  return { success: true, applied };
}
