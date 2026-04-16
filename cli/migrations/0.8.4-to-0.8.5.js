/**
 * Migration: 0.8.4 → 0.8.5
 *
 * Session-scoped task namespace.
 *
 * - Clear .mint/sessions/*.json (old self-generated ids no longer match Claude session_id)
 *   while preserving .mint/sessions/.current-session-id (written by the session hook).
 * - Quarantine pre-existing .mint/tasks/<feature>/ entries under
 *   .mint/tasks/_legacy-pre-0.8.5/<feature>/ as a user escape hatch.
 * - Re-run ensureGitignore so the (already covering) .mint/tasks/ entry is present.
 *
 * Idempotent: safe to re-run (skips _legacy-pre-0.8.5/ itself, no-ops on missing dirs).
 */
import fs from 'fs';
import path from 'path';
import { ensureGitignore } from '../lib/gitignore.js';

const LEGACY_DIR = '_legacy-pre-0.8.5';

function clearOldSessions(projectRoot) {
  const sessionsDir = path.join(projectRoot, '.mint', 'sessions');
  if (!fs.existsSync(sessionsDir)) return;

  let entries;
  try {
    entries = fs.readdirSync(sessionsDir);
  } catch (err) {
    process.stderr.write(`[migration] failed to read sessions dir ${sessionsDir}: ${err.message}\n`);
    return;
  }

  for (const entry of entries) {
    // Preserve .current-session-id (and any other dot-prefixed metadata files)
    if (entry.startsWith('.')) continue;
    if (!entry.endsWith('.json')) continue;

    const filePath = path.join(sessionsDir, entry);
    try {
      const stat = fs.lstatSync(filePath);
      if (stat.isFile()) fs.unlinkSync(filePath);
    } catch (err) {
      // best-effort cleanup, but surface failures so users notice stuck files
      process.stderr.write(`[migration] failed to delete session file ${filePath}: ${err.message}\n`);
    }
  }
}

function moveDir(src, dest) {
  try {
    fs.renameSync(src, dest);
  } catch (err) {
    if (err && err.code === 'EXDEV') {
      fs.cpSync(src, dest, { recursive: true });
      fs.rmSync(src, { recursive: true, force: true });
    } else {
      throw err;
    }
  }
}

function quarantineLegacyTasks(projectRoot) {
  const tasksDir = path.join(projectRoot, '.mint', 'tasks');
  if (!fs.existsSync(tasksDir)) return;

  let entries;
  try {
    entries = fs.readdirSync(tasksDir);
  } catch (err) {
    process.stderr.write(`[migration] failed to read tasks dir ${tasksDir}: ${err.message}\n`);
    return;
  }

  // Filter to direntries that are not the quarantine bucket itself.
  const candidates = entries.filter(name => name !== LEGACY_DIR);
  if (candidates.length === 0) return;

  const legacyRoot = path.join(tasksDir, LEGACY_DIR);
  // Refuse to write through a symlinked quarantine root — could escape the project tree.
  if (fs.existsSync(legacyRoot)) {
    let legacyStat;
    try {
      legacyStat = fs.lstatSync(legacyRoot);
    } catch (err) {
      process.stderr.write(`[migration] failed to stat ${legacyRoot}: ${err.message}\n`);
      return;
    }
    if (legacyStat.isSymbolicLink()) {
      process.stderr.write(`[migration] ${legacyRoot}: quarantine root is a symlink (skipping migration)\n`);
      return;
    }
  } else {
    fs.mkdirSync(legacyRoot, { recursive: true });
  }

  for (const name of candidates) {
    const src = path.join(tasksDir, name);
    const dest = path.join(legacyRoot, name);

    // Reject symlinked entries — renaming/copying through them could touch files
    // outside the project tree.
    let srcStat;
    try {
      srcStat = fs.lstatSync(src);
    } catch (err) {
      process.stderr.write(`[migration] failed to stat ${src}: ${err.message}\n`);
      continue;
    }
    if (srcStat.isSymbolicLink()) {
      process.stderr.write(`[migration] ${name}: symlink entry in .mint/tasks/ (skipping)\n`);
      continue;
    }

    // If already moved (idempotent re-run), skip — but tell the user so they don't
    // assume a partial-failure rerun cleaned everything up.
    if (fs.existsSync(dest)) {
      process.stderr.write(`[migration] ${name}: already quarantined (skipping)\n`);
      continue;
    }
    if (!fs.existsSync(src)) continue;

    moveDir(src, dest);
  }
}

function updateGitignore(projectRoot) {
  // Existing `.mint/tasks/` entry already covers `.mint/tasks/_legacy-pre-0.8.5/`.
  // Re-run to guarantee the entry is present even on projects that pruned it.
  ensureGitignore(projectRoot);
}

export default {
  from: '0.8.4',
  to: '0.8.5',
  description: 'Session-scoped task namespace',
  steps: [
    { name: 'Clear old session files', fn: clearOldSessions },
    { name: 'Quarantine legacy tasks', fn: quarantineLegacyTasks },
    { name: 'Ensure gitignore covers legacy path', fn: updateGitignore },
  ],
};
