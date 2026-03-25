/**
 * Git worktree management for parallel spec execution.
 *
 * Creates isolated worktrees at .mint/worktrees/<slug>/ for each parallel spec.
 * Handles creation, env propagation, and cleanup.
 */
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

/**
 * Create a git worktree for a spec.
 * @param {string} projectRoot - absolute path to the project root
 * @param {string} slug - unique name for the worktree (e.g., "spec-001")
 * @param {object} options
 * @param {string} [options.baseBranch] - branch to base off (default: current HEAD)
 * @returns {{ worktreePath: string, branch: string }}
 */
export function createWorktree(projectRoot, slug, options = {}) {
  const worktreeDir = path.join(projectRoot, '.mint', 'worktrees');
  const worktreePath = path.join(worktreeDir, slug);
  const branch = `mint/${slug}`;

  // Ensure .mint/worktrees/ exists
  fs.mkdirSync(worktreeDir, { recursive: true });

  // Remove stale worktree if it exists
  if (fs.existsSync(worktreePath)) {
    try {
      execSync(`git worktree remove "${worktreePath}" --force`, { cwd: projectRoot, stdio: 'pipe' });
    } catch {
      // Force remove directory if git worktree remove fails
      fs.rmSync(worktreePath, { recursive: true, force: true });
      execSync('git worktree prune', { cwd: projectRoot, stdio: 'pipe' });
    }
  }

  // Delete branch if it exists from a previous run
  try {
    execSync(`git branch -D "${branch}"`, { cwd: projectRoot, stdio: 'pipe' });
  } catch { /* branch doesn't exist, fine */ }

  // Create worktree
  const base = options.baseBranch || 'HEAD';
  execSync(`git worktree add "${worktreePath}" -b "${branch}" ${base}`, {
    cwd: projectRoot,
    stdio: 'pipe',
  });

  // Copy untracked env files into worktree
  copyEnvFiles(projectRoot, worktreePath);

  return { worktreePath, branch };
}

/**
 * Remove a worktree and its branch.
 * @param {string} projectRoot
 * @param {string} slug
 */
export function removeWorktree(projectRoot, slug) {
  const worktreePath = path.join(projectRoot, '.mint', 'worktrees', slug);
  const branch = `mint/${slug}`;

  try {
    execSync(`git worktree remove "${worktreePath}" --force`, { cwd: projectRoot, stdio: 'pipe' });
  } catch {
    if (fs.existsSync(worktreePath)) {
      fs.rmSync(worktreePath, { recursive: true, force: true });
    }
    try { execSync('git worktree prune', { cwd: projectRoot, stdio: 'pipe' }); } catch { /* ignore */ }
  }

  // Clean up branch
  try {
    execSync(`git branch -D "${branch}"`, { cwd: projectRoot, stdio: 'pipe' });
  } catch { /* branch doesn't exist */ }
}

/**
 * List all active mint worktrees.
 * @param {string} projectRoot
 * @returns {Array<{ slug: string, path: string, branch: string }>}
 */
export function listWorktrees(projectRoot) {
  const worktreeDir = path.join(projectRoot, '.mint', 'worktrees');
  if (!fs.existsSync(worktreeDir)) return [];

  try {
    const entries = fs.readdirSync(worktreeDir, { withFileTypes: true });
    return entries
      .filter(e => e.isDirectory())
      .map(e => ({
        slug: e.name,
        path: path.join(worktreeDir, e.name),
        branch: `mint/${e.name}`,
      }));
  } catch {
    return [];
  }
}

/**
 * Remove all mint worktrees (cleanup command).
 * @param {string} projectRoot
 * @returns {number} count of removed worktrees
 */
export function cleanAllWorktrees(projectRoot) {
  const worktrees = listWorktrees(projectRoot);
  let count = 0;
  for (const wt of worktrees) {
    removeWorktree(projectRoot, wt.slug);
    count++;
  }
  // Final prune
  try { execSync('git worktree prune', { cwd: projectRoot, stdio: 'pipe' }); } catch { /* ignore */ }
  return count;
}

/**
 * Copy .env files from project root to worktree.
 * Copies: .env, .env.local, .env.development, .env.development.local
 * @param {string} projectRoot
 * @param {string} worktreePath
 */
function copyEnvFiles(projectRoot, worktreePath) {
  const envPatterns = ['.env', '.env.local', '.env.development', '.env.development.local', '.env.test'];
  for (const pattern of envPatterns) {
    const src = path.join(projectRoot, pattern);
    const dest = path.join(worktreePath, pattern);
    if (fs.existsSync(src)) {
      try { fs.copyFileSync(src, dest); } catch { /* skip if can't copy */ }
    }
  }
}

/**
 * Merge a worktree's changes back to the base branch.
 * Collects all commits from the worktree branch and cherry-picks them.
 * @param {string} projectRoot
 * @param {string} slug
 * @param {string} baseBranch - branch to merge into
 * @returns {{ merged: boolean, commits: number, conflicts: boolean }}
 */
export function mergeWorktree(projectRoot, slug, baseBranch) {
  const branch = `mint/${slug}`;

  try {
    // Get commits unique to the worktree branch
    const log = execSync(`git log ${baseBranch}..${branch} --oneline`, {
      cwd: projectRoot, encoding: 'utf8', stdio: 'pipe',
    }).trim();

    if (!log) return { merged: true, commits: 0, conflicts: false };

    const commitCount = log.split('\n').length;

    // Try merge (no fast-forward to keep history clean)
    try {
      execSync(`git merge ${branch} --no-ff -m "merge: ${slug}"`, {
        cwd: projectRoot, stdio: 'pipe',
      });
      return { merged: true, commits: commitCount, conflicts: false };
    } catch {
      // Merge conflict — abort and report
      try { execSync('git merge --abort', { cwd: projectRoot, stdio: 'pipe' }); } catch { /* ignore */ }
      return { merged: false, commits: commitCount, conflicts: true };
    }
  } catch {
    return { merged: false, commits: 0, conflicts: false };
  }
}
