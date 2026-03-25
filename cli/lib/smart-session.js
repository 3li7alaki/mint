/**
 * Smart session — spawn a claude -p session to operate on a project intelligently.
 *
 * Used by: mint doctor --smart, mint update --smart, mint init --smart, mint optimize
 * Instead of generic scripts, these spawn Claude to read the actual project, make
 * context-aware decisions, and APPLY the changes itself.
 */
import { spawn } from 'child_process';

/**
 * Spawn a claude -p session that can both analyze AND modify a project.
 *
 * @param {string} cwd - project root
 * @param {string} prompt - task to perform
 * @param {object} options
 * @param {number} [options.maxTurns=30] - max agentic turns
 * @param {number} [options.timeoutMs=180000] - timeout (3 min default)
 * @param {string} [options.model='sonnet'] - model to use
 * @param {boolean} [options.readOnly=false] - if true, only allow Read/Glob/Grep/Bash
 * @returns {Promise<{ success: boolean, result: string, error?: string }>}
 */
export function smartSession(cwd, prompt, options = {}) {
  const {
    maxTurns = 30,
    timeoutMs = 180_000,
    model = 'sonnet',
    readOnly = false,
  } = options;

  const allowedTools = readOnly
    ? ['Read', 'Glob', 'Grep', 'Bash']
    : ['Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash'];

  return new Promise((resolve, reject) => {
    const args = [
      '-p', prompt,
      '--output-format', 'json',
      '--max-turns', String(maxTurns),
      '--model', model,
      '--allowedTools', allowedTools.join(','),
    ];

    const proc = spawn('claude', args, {
      cwd,
      env: { ...process.env },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (data) => { stdout += data.toString(); });
    proc.stderr.on('data', (data) => { stderr += data.toString(); });

    const timer = setTimeout(() => {
      proc.kill('SIGTERM');
      resolve({ success: false, result: '', error: `Session timed out after ${timeoutMs / 1000}s` });
    }, timeoutMs);

    proc.on('close', (code) => {
      clearTimeout(timer);
      try {
        const output = JSON.parse(stdout);
        resolve({
          success: code === 0,
          result: output.result || '',
          sessionId: output.session_id || null,
        });
      } catch {
        resolve({
          success: code === 0,
          result: stdout.slice(0, 5000),
          error: code !== 0 ? stderr.slice(0, 1000) : null,
        });
      }
    });

    proc.on('error', (err) => {
      clearTimeout(timer);
      resolve({ success: false, result: '', error: err.message });
    });
  });
}

/**
 * Smart doctor — analyze AND fix a project's mint config.
 * Claude reads the codebase, understands the stack, and applies the right config.
 */
export function smartDoctor(projectRoot) {
  return smartSession(projectRoot, `You are a mint configuration expert. Analyze this project and FIX its .mint/config.json to match what the project actually needs.

DO NOT just recommend — actually edit .mint/config.json with the correct values.

Steps:
1. Read package.json (or equivalent) to understand the stack, deps, scripts
2. Glob for test files, config files, UI files to understand the project structure
3. Read .mint/config.json to see current configuration
4. Edit .mint/config.json to fix:
   - gates.lint: set to the actual lint command (from package.json scripts or eslint/biome config)
   - gates.types: set to the actual type check command (tsc --noEmit, etc.)
   - gates.tests: set to the actual test command (from package.json scripts)
   - stack: set to the detected framework
   - packageManager: set to the detected PM (check lockfiles)
   - browser.enabled: true if project has UI files
   - design.enabled: true if project has UI files (.tsx/.jsx/.vue/.svelte/.css)
   - reviewers: enable security if project handles auth/payments, enable tests if project has tests
5. Check .gitignore for missing mint entries and add them
6. Check if JSONL learning files exist (.mint/issues.jsonl etc.), create if missing

Report what you changed. Be concise.`, {
    maxTurns: 20,
    model: 'sonnet',
    readOnly: false, // Can edit files
  });
}

/**
 * Smart update — migrate a project's config with project-aware intelligence.
 */
export function smartUpdate(projectRoot, fromVersion, toVersion) {
  return smartSession(projectRoot, `You are updating a mint project from v${fromVersion} to v${toVersion}.

Read the project and .mint/config.json, then APPLY the right changes:

1. Read the codebase to understand what this project is
2. Add any missing config keys with values that make sense for THIS project (not generic defaults)
3. If old .md learning files exist (issues.md, wins.md, etc.) and .jsonl versions don't, create the .jsonl files
4. Update .gitignore if missing mint entries
5. Any other migration needed for ${toVersion}

Actually make the changes — don't just list recommendations. Report what you did.`, {
    maxTurns: 15,
    model: 'sonnet',
    readOnly: false,
  });
}

/**
 * Smart init — set up mint with full project understanding.
 * Way better than the generic detection in init.js.
 */
export function smartInit(projectRoot) {
  return smartSession(projectRoot, `You are setting up mint for this project. Analyze the codebase thoroughly and create the perfect .mint/config.json.

Steps:
1. Read package.json, lockfiles, framework configs to understand the full stack
2. Find the test runner (check package.json scripts, vitest.config, jest.config, etc.)
3. Find the linter (eslint, biome, etc.) and its run command
4. Find the type checker (tsc, pyright, etc.)
5. Check for UI files to decide on design intelligence
6. Check git log --oneline -5 to see if solo or collaborative repo
7. Create .mint/ directory with:
   - config.json — fully configured for this specific project
   - hard-blocks.md — with project-specific constraints if obvious
   - issues.jsonl, wins.jsonl, patterns.jsonl, instincts.jsonl (empty)
8. Update .gitignore with mint entries
9. Create or update CLAUDE.md with mint section

The config should be SPECIFIC to this project — not generic defaults. If the project uses
vitest, the test command should be "vitest run", not "bun test". If it uses pnpm, set pnpm.
If it has 50+ Vue files, enable design. If it's one contributor, set repoMode to "solo".

Report what you configured and why.`, {
    maxTurns: 25,
    model: 'sonnet',
    readOnly: false,
  });
}
