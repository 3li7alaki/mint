import * as p from '@clack/prompts';
import path from 'path';
import fs from 'fs';
import { execSync } from 'child_process';
import { readJsonSafe, fileExists, detectStack, detectTool, detectContextMode, getGlobalConfigPath, loadGlobalConfig, GLOBAL_KEYS, ensureReadmeSignature } from '../lib/detect.js';
import { spinner } from '../lib/spinner.js';

export async function run(flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');
  const autoFix = flags.fix || false;
  const quick = flags.quick || false;

  p.intro(`\x1b[32m mint doctor${autoFix ? ' --fix' : ''}${quick ? ' --quick' : ''} \x1b[0m`);

  const critical = [];  // Blocks work
  const warns = [];     // Degrades quality
  const info = [];      // Suggestions
  let fixed = 0;

  function addCritical(msg, fix) { critical.push({ msg, fix }); }
  function addWarn(msg, fix) { warns.push({ msg, fix }); }
  function addInfo(msg) { info.push(msg); }

  function tryFix(issue) {
    if (!autoFix || !issue.fix) return false;
    try {
      issue.fix();
      fixed++;
      return true;
    } catch {
      return false;
    }
  }

  // ─── Critical checks (blocks work) ──────────────────────────────────────

  // Config
  if (!fileExists(configPath) || !readJsonSafe(configPath)) {
    addCritical('.mint/config.json missing or invalid — run mint init');
  }

  const config = readJsonSafe(configPath) || {};

  // Hard blocks
  if (!fileExists(path.join(mintDir, 'hard-blocks.md'))) {
    addCritical('.mint/hard-blocks.md missing', () => {
      fs.writeFileSync(path.join(mintDir, 'hard-blocks.md'),
        '# Hard Blocks — What Agents Can NEVER Do\n\n## Universal\n- NEVER `git push`\n- NEVER modify files outside declared task scope\n- NEVER fix bad output directly — reset and fix the spec\n- NEVER continue after 2 failures on the same spec\n');
    });
  }

  // Gates (only run in non-quick mode)
  if (!quick && config.gates) {
    for (const [name, cmd] of Object.entries(config.gates)) {
      if (!cmd || cmd === false) continue;
      const command = typeof cmd === 'string' ? cmd : cmd.command;
      if (!command) continue;
      try {
        execSync(command, { cwd, stdio: 'ignore', timeout: 30000 });
      } catch {
        addCritical(`Gate: ${name} fails — \`${command}\`\n    Fix: Run the command manually to see full output`);
      }
    }
  }

  // Bun
  if (!detectTool('bun')) {
    addCritical('Bun not installed — mint CLI requires Bun\n    Fix: curl -fsSL https://bun.sh/install | bash');
  }

  // Git
  if (!fileExists(path.join(cwd, '.git'))) {
    addCritical('Not a git repository — mint requires git');
  }

  // ─── Warning checks (degrades quality) ──────────────────────────────────

  // Learning files — JSONL format (new) with backwards compat for .md (old)
  const learningFiles = [
    { name: 'issues', ext: 'jsonl' },
    { name: 'wins', ext: 'jsonl' },
    { name: 'patterns', ext: 'jsonl' },
    { name: 'instincts', ext: 'jsonl' },
  ];
  for (const { name, ext } of learningFiles) {
    const jsonlPath = path.join(mintDir, `${name}.${ext}`);
    const mdPath = path.join(mintDir, `${name}.md`);
    if (fileExists(jsonlPath)) {
      // Good — using new format
    } else if (fileExists(mdPath)) {
      addWarn(`${name}.md exists but ${name}.jsonl doesn't — migrate to JSONL for concurrent-safe appending`, () => {
        fs.writeFileSync(jsonlPath, '');
        // Note: actual data migration happens via cli/lib/jsonl.js migrateMarkdownTableToJsonl
      });
    } else {
      addWarn(`.mint/${name}.${ext} missing`, () => {
        fs.writeFileSync(jsonlPath, '');
      });
    }
  }

  // Browser
  if (config.browser?.enabled) {
    if (!detectTool('pinchtab')) {
      addWarn('Browser enabled but PinchTab not installed\n    Fix: curl -fsSL https://pinchtab.com/install.sh | sh');
    }
  }

  // Context Mode
  if (config.context?.enabled) {
    if (!detectContextMode()) {
      addWarn('Context Mode enabled but context-mode not installed\n    Fix: claude mcp add context-mode -- npx -y context-mode');
    }
  }

  // Design
  if (config.design?.enabled) {
    const profilePath = config.design.profile
      ? path.join(cwd, config.design.profile)
      : path.join(mintDir, 'design-profile.json');
    if (!fileExists(profilePath)) {
      addWarn('Design enabled but no profile built\n    Fix: Run /design:profile build after you have UI code');
    }
  }

  // Reviewers
  const importantReviewers = ['spec', 'quality', 'conventions'];
  for (const name of importantReviewers) {
    const rev = config.reviewers?.[name];
    if (!rev || rev === false || (typeof rev === 'object' && !rev.enabled)) {
      addWarn(`Reviewer disabled: ${name}\n    Why: ${name === 'spec' ? 'Spec compliance is the stage 1 gate' : name === 'security' ? 'OWASP top 10 checks won\'t run' : 'Quality checks skipped'}`);
    }
  }

  // CLAUDE.md
  const claudeMdPath = path.join(cwd, 'CLAUDE.md');
  if (fileExists(claudeMdPath)) {
    const claudeMd = fs.readFileSync(claudeMdPath, 'utf8');
    if (!claudeMd.includes('<!-- mint:start')) {
      addWarn('CLAUDE.md missing mint section — run mint init to add');
    }
  } else {
    addWarn('CLAUDE.md not found — run mint init to create');
  }

  // Doc-manifest
  const manifestPath = path.join(mintDir, 'doc-manifest.json');
  if (fileExists(manifestPath)) {
    const manifest = readJsonSafe(manifestPath);
    if (manifest?.docs) {
      const emptySections = manifest.docs.filter(d => !d.sections || d.sections.length === 0);
      if (emptySections.length > 0) {
        addWarn(`Doc-manifest: ${emptySections.length} docs with no tracked sections\n    Fix: Run /doc-setup to populate`);
      }
    }
  }

  // .gitignore completeness
  {
    const { MINT_IGNORE_ENTRIES, ensureGitignore: ensureGI } = await import('../lib/gitignore.js');
    const gitignorePath = path.join(cwd, '.gitignore');
    if (fileExists(gitignorePath)) {
      const gi = fs.readFileSync(gitignorePath, 'utf8');
      const missing = MINT_IGNORE_ENTRIES.filter(i => !gi.includes(i));
      if (missing.length > 0) {
        addWarn(`.gitignore missing ${missing.length} mint entries`, () => {
          ensureGI(cwd);
        });
      }
    }
  }

  // ─── Info checks (suggestions) ──────────────────────────────────────────

  // Global config
  const globalConfig = loadGlobalConfig();
  if (Object.keys(globalConfig).length === 0) {
    addInfo('No global config — set user defaults with: mint config set --global');
  }

  // Stack detection
  const detected = detectStack(cwd);
  if (config.stack && detected !== 'none' && config.stack !== detected) {
    addInfo(`Stack mismatch: config=${config.stack}, detected=${detected}`);
  }

  // Claude CLI
  if (!detectTool('claude')) {
    addInfo('Claude CLI not found — plugin features won\'t work');
  }

  // Signature (only inform if undefined, never nag if false)
  if (config.signature === undefined) {
    addInfo('Signature not configured — set config.signature to true/false');
  }

  // ─── Auto-fix ───────────────────────────────────────────────────────────

  if (autoFix) {
    for (const issue of [...critical, ...warns]) {
      tryFix(issue);
    }
  }

  // ─── Output (tiered) ───────────────────────────────────────────────────

  if (critical.length > 0) {
    console.log(`\n  \x1b[31mCRITICAL (blocks work)\x1b[0m`);
    for (const issue of critical) {
      const fixedTag = autoFix && issue.fix ? ' \x1b[34m— fixed\x1b[0m' : '';
      console.log(`  \x1b[31m✗\x1b[0m ${issue.msg}${fixedTag}`);
    }
  }

  if (warns.length > 0) {
    console.log(`\n  \x1b[33mWARNINGS (degrades quality)\x1b[0m`);
    for (const issue of warns) {
      const fixedTag = autoFix && issue.fix ? ' \x1b[34m— fixed\x1b[0m' : '';
      console.log(`  \x1b[33m⚠\x1b[0m ${issue.msg}${fixedTag}`);
    }
  }

  if (info.length > 0) {
    console.log(`\n  \x1b[2mINFO (suggestions)\x1b[0m`);
    for (const msg of info) {
      console.log(`  \x1b[2m○\x1b[0m ${msg}`);
    }
  }

  if (critical.length === 0 && warns.length === 0) {
    console.log(`\n  \x1b[32m✓ All checks passed\x1b[0m`);
  }

  // Summary
  const parts = [];
  if (critical.length) parts.push(`\x1b[31m${critical.length} critical\x1b[0m`);
  if (warns.length) parts.push(`\x1b[33m${warns.length} warnings\x1b[0m`);
  if (info.length) parts.push(`\x1b[2m${info.length} info\x1b[0m`);
  if (fixed) parts.push(`\x1b[34m${fixed} fixed\x1b[0m`);

  p.outro(parts.join(', ') || '\x1b[32mhealthy\x1b[0m');

  // ─── Smart analysis — Claude reads the project and applies fixes ─────────
  // Always runs when Claude is available. This is how mint works.

  if (detectTool('claude') && autoFix && !quick) {
    console.log('');
    const s = spinner();
    s.start('Running smart analysis (claude -p)...');
    try {
      const { smartDoctor } = await import('../lib/smart-session.js');
      const analysis = await smartDoctor(cwd);
      s.stop('Smart analysis complete');
      if (analysis.success && analysis.result) {
        console.log(`\n  \x1b[1mSmart recommendations\x1b[0m\n`);
        // Indent each line of the result
        for (const line of analysis.result.split('\n')) {
          if (line.trim()) console.log(`  ${line}`);
        }
        console.log('');
      } else {
        s.stop('Smart analysis returned no recommendations');
      }
    } catch (err) {
      s.stop(`Smart analysis failed: ${err.message}`);
    }
  }
}
