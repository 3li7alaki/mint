import * as p from '@clack/prompts';
import { cleanAllWorktrees, listWorktrees } from '../lib/worktree.js';

export async function run(args = [], flags = {}) {
  const cwd = process.cwd();

  p.intro('\x1b[32m mint clean \x1b[0m');

  const worktrees = listWorktrees(cwd);

  if (worktrees.length === 0) {
    p.log.info('No mint worktrees found.');
    p.outro('Clean');
    return;
  }

  p.log.info(`Found ${worktrees.length} worktree${worktrees.length > 1 ? 's' : ''}:`);
  for (const wt of worktrees) {
    console.log(`  ${wt.slug} → ${wt.path}`);
  }

  if (!flags.yes) {
    const confirm = await p.confirm({
      message: `Remove all ${worktrees.length} worktrees?`,
      initialValue: true,
    });
    if (p.isCancel(confirm) || !confirm) {
      p.outro('Cancelled');
      return;
    }
  }

  const count = cleanAllWorktrees(cwd);
  p.log.success(`Removed ${count} worktree${count > 1 ? 's' : ''}`);
  p.outro('Clean');
}
