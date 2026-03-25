/**
 * Structured error formatting for mint CLI.
 *
 * Every error should have: what failed, why (exit code + output), and what to do next.
 */

/**
 * Format a command execution error with actionable guidance.
 * @param {object} opts
 * @param {string} opts.action - what was being done ("PinchTab install", "lint gate")
 * @param {string} opts.command - the command that was run
 * @param {number|null} opts.exitCode - process exit code
 * @param {string} opts.output - stderr or stdout from the failed command
 * @param {string[]} opts.causes - possible causes
 * @param {string[]} opts.fixes - suggested fixes
 * @returns {string}
 */
export function formatError({ action, command, exitCode, output, causes = [], fixes = [] }) {
  const lines = [];

  lines.push(`\x1b[31mError: ${action} failed\x1b[0m`);
  lines.push('');

  if (command) {
    lines.push(`  Command: \x1b[2m${command}\x1b[0m`);
  }
  if (exitCode !== null && exitCode !== undefined) {
    lines.push(`  Exit code: ${exitCode}`);
  }
  if (output) {
    const trimmed = output.trim().split('\n').slice(0, 5).join('\n');
    lines.push(`  Output: \x1b[2m${trimmed}\x1b[0m`);
  }

  if (causes.length > 0) {
    lines.push('');
    lines.push('  Possible causes:');
    for (const cause of causes) {
      lines.push(`  \x1b[33m*\x1b[0m ${cause}`);
    }
  }

  if (fixes.length > 0) {
    lines.push('');
    lines.push('  Try:');
    for (const fix of fixes) {
      lines.push(`  \x1b[36m>\x1b[0m ${fix}`);
    }
  }

  lines.push('');
  return lines.join('\n');
}

/**
 * Wrap an execSync call with structured error handling.
 * @param {string} command
 * @param {object} execOptions - options for execSync
 * @param {object} errorContext - context for error formatting
 * @param {string} errorContext.action
 * @param {string[]} [errorContext.causes]
 * @param {string[]} [errorContext.fixes]
 * @returns {{ success: boolean, output?: string, error?: string }}
 */
export function safeExec(command, execOptions, errorContext) {
  const { execSync } = await_import_child_process();
  try {
    const output = require('child_process').execSync(command, {
      encoding: 'utf8',
      stdio: 'pipe',
      ...execOptions,
    });
    return { success: true, output: output.trim() };
  } catch (err) {
    const formatted = formatError({
      action: errorContext.action,
      command,
      exitCode: err.status,
      output: err.stderr || err.stdout || '',
      causes: errorContext.causes || [],
      fixes: errorContext.fixes || [],
    });
    return { success: false, error: formatted };
  }
}

// Lazy import to avoid top-level require in ESM
function await_import_child_process() {
  return { execSync: require('child_process').execSync };
}

/**
 * Common error templates for frequently occurring failures.
 */
export const ERROR_TEMPLATES = {
  pinchtabInstall: {
    action: 'PinchTab install',
    causes: [
      'No internet connection',
      'DNS resolution failure',
      'Firewall blocking the request',
    ],
    fixes: [
      'Check your internet connection',
      'Install PinchTab manually: https://github.com/pinchtab/pinchtab',
      'Skip browser setup: mint init --browser false',
    ],
  },
  contextModeInstall: {
    action: 'context-mode install',
    causes: [
      'Claude CLI not installed',
      'MCP server registration failed',
    ],
    fixes: [
      'Install Claude CLI first',
      'Manual install: claude mcp add context-mode -- npx -y context-mode',
      'Skip context: mint init --context false',
    ],
  },
  gateFailure: (gate) => ({
    action: `${gate} gate`,
    causes: [
      `${gate} command not found or not configured`,
      'Code has errors',
      'Dependencies not installed',
    ],
    fixes: [
      `Run the ${gate} command manually to see full output`,
      'Check mint config: mint config',
      'Run: mint doctor --fix',
    ],
  }),
  bunNotFound: {
    action: 'Bun detection',
    causes: [
      'Bun is not installed',
      'Bun is not in PATH',
    ],
    fixes: [
      'Install Bun: curl -fsSL https://bun.sh/install | bash',
      'Add Bun to PATH: export PATH="$HOME/.bun/bin:$PATH"',
    ],
  },
};
