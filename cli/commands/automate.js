import fs from 'fs';
import path from 'path';
import { readJsonSafe } from '../lib/detect.js';
import { readJsonl, formatTable } from '../lib/jsonl.js';

export async function run(args = [], flags = {}) {
  const cwd = process.cwd();
  const mintDir = path.join(cwd, '.mint');
  const configPath = path.join(mintDir, 'config.json');

  const config = readJsonSafe(configPath);
  if (!config) {
    console.log('\n  \x1b[31mNo mint config found.\x1b[0m Run \x1b[36mmint init\x1b[0m\n');
    return;
  }

  const c = (v) => `\x1b[36m${v}\x1b[0m`;
  const g = (v) => `\x1b[32m${v}\x1b[0m`;
  const y = (v) => `\x1b[33m${v}\x1b[0m`;
  const r = (v) => `\x1b[31m${v}\x1b[0m`;
  const d = (v) => `\x1b[2m${v}\x1b[0m`;

  const sub = args[0];

  if (sub === 'generate') {
    return showGenerate(args[1], mintDir, { c, g, y, r, d });
  }

  if (sub === 'promote') {
    return runPromote(args[1], cwd, mintDir, { c, g, y, r, d });
  }

  if (sub === 'record') {
    return flagRecord(mintDir, { c, g, d });
  }

  if (sub === 'skills') {
    return listSkills(mintDir, { c, g, y, d });
  }

  // Default / explicit 'list': show workflow candidates
  return listCandidates(mintDir, { c, g, y, d });
}

function listCandidates(mintDir, { c, g, y, d }) {
  const candidatesPath = path.join(mintDir, 'workflow-candidates.jsonl');
  const candidates = readJsonl(candidatesPath);

  if (candidates.length === 0) {
    console.log('\n  No workflow candidates detected yet.');
    console.log(`  ${d('Candidates appear after repeated workflow patterns are observed.')}\n`);
    return;
  }

  console.log(`\n  \x1b[1mWorkflow Candidates\x1b[0m ${d(`(${candidates.length} total)`)}\n`);

  const table = formatTable(candidates, ['proposedName', 'occurrences', 'confidence', 'trustLevel', 'fingerprint'], {
    labels: {
      proposedName: 'NAME',
      occurrences: 'OCC',
      confidence: 'CONF',
      trustLevel: 'TRUST',
      fingerprint: 'FINGERPRINT',
    },
    widths: { proposedName: 25, fingerprint: 40 },
  });

  console.log(table);
  console.log('');
  console.log(`  ${d('Commands:')}`);
  console.log(`    ${c('mint automate generate <name>')}  ${d('Generate a skill from a candidate')}`);
  console.log(`    ${c('mint automate promote <name>')}   ${d('Copy generated skill to .claude/skills/')}`);
  console.log(`    ${c('mint automate skills')}           ${d('List generated skills')}`);
  console.log('');
}

function showGenerate(name, mintDir, { c, g, y, r, d }) {
  if (!name) {
    console.log(`\n  ${r('Usage:')} mint automate generate <name>\n`);
    return;
  }

  const candidatesPath = path.join(mintDir, 'workflow-candidates.jsonl');
  const candidates = readJsonl(candidatesPath);
  const candidate = candidates.find(c => c.proposedName === name);

  if (!candidate) {
    console.log(`\n  ${r('No candidate found with name:')} ${name}`);
    console.log(`  ${d('Run')} ${c('mint automate')} ${d('to see available candidates.')}\n`);
    return;
  }

  console.log(`\n  \x1b[1mCandidate: ${name}\x1b[0m\n`);
  console.log(`  Fingerprint:  ${c(candidate.fingerprint)}`);
  console.log(`  Occurrences:  ${c(candidate.occurrences || 0)}`);
  console.log(`  Confidence:   ${c(candidate.confidence || 0)}`);
  console.log(`  Trust level:  ${c(candidate.trustLevel || 0)}`);
  console.log(`  First seen:   ${d(candidate.firstSeen || 'unknown')}`);
  console.log(`  Last seen:    ${d(candidate.lastSeen || 'unknown')}`);

  if (candidate.variableSlots && candidate.variableSlots.length > 0) {
    console.log(`  Variables:    ${candidate.variableSlots.join(', ')}`);
  }

  console.log('');
  console.log(`  ${d('To generate a skill from this candidate, tell Claude:')}`);
  console.log(`  ${c(`"generate skill from workflow candidate ${name}"`)}`);
  console.log('');
}

function runPromote(name, cwd, mintDir, { c, g, y, r, d }) {
  if (!name) {
    console.log(`\n  ${r('Usage:')} mint automate promote <name>\n`);
    return;
  }

  const srcDir = path.join(mintDir, 'skills', name);
  if (!fs.existsSync(srcDir)) {
    console.log(`\n  ${r('No generated skill found:')} .mint/skills/${name}/`);
    console.log(`  ${d('Run')} ${c('mint automate skills')} ${d('to see available skills.')}\n`);
    return;
  }

  const destDir = path.join(cwd, '.claude', 'skills', name);

  // Check if target already exists
  if (fs.existsSync(destDir)) {
    console.log(`\n  ${r('Skill already exists at:')} .claude/skills/${name}/`);
    console.log(`  ${d('Remove it first if you want to re-promote.')}\n`);
    return;
  }

  // Create .claude/skills/ if needed
  fs.mkdirSync(path.dirname(destDir), { recursive: true });

  // Copy the skill directory
  copyDirSync(srcDir, destDir);

  // Update SKILL.md frontmatter — remove disable-model-invocation
  const skillMdPath = path.join(destDir, 'SKILL.md');
  if (fs.existsSync(skillMdPath)) {
    let content = fs.readFileSync(skillMdPath, 'utf8');
    // Remove the disable-model-invocation line from frontmatter
    content = content.replace(/^disable-model-invocation:\s*true\n?/m, '');
    content = content.replace(/^disable-model-invocation:\s*false\n?/m, '');
    fs.writeFileSync(skillMdPath, content);
  }

  console.log(`\n  ${g('Promoted:')} .mint/skills/${name}/ -> .claude/skills/${name}/`);
  if (fs.existsSync(skillMdPath)) {
    console.log(`  ${g('Updated:')} Removed disable-model-invocation from SKILL.md`);
  }
  console.log('');
}

function flagRecord(mintDir, { c, g, d }) {
  const sessionDir = path.join(mintDir, 'sessions');
  if (!fs.existsSync(sessionDir)) {
    fs.mkdirSync(sessionDir, { recursive: true });
  }

  const flagPath = path.join(sessionDir, 'recording-active');
  fs.writeFileSync(flagPath, JSON.stringify({ startedAt: new Date().toISOString() }) + '\n');

  console.log(`\n  ${g('Recording active.')} This session will be tracked for workflow detection.`);
  console.log(`  ${d('The flag is stored at .mint/sessions/recording-active')}\n`);
}

function listSkills(mintDir, { c, g, y, d }) {
  const skillsDir = path.join(mintDir, 'skills');

  if (!fs.existsSync(skillsDir)) {
    console.log('\n  No generated skills yet.');
    console.log(`  ${d('Skills are generated from workflow candidates.')}\n`);
    return;
  }

  let entries;
  try {
    entries = fs.readdirSync(skillsDir, { withFileTypes: true })
      .filter(e => e.isDirectory());
  } catch {
    entries = [];
  }

  if (entries.length === 0) {
    console.log('\n  No generated skills yet.');
    console.log(`  ${d('Skills are generated from workflow candidates.')}\n`);
    return;
  }

  console.log(`\n  \x1b[1mGenerated Skills\x1b[0m ${d(`(${entries.length} total)`)}\n`);
  console.log(`  ${'NAME'.padEnd(25)}${'TRUST'.padEnd(8)}${'STATUS'}`);
  console.log(`  ${'─'.repeat(50)}`);

  for (const entry of entries) {
    const name = entry.name;
    const skillMdPath = path.join(skillsDir, name, 'SKILL.md');
    let trustLevel = '?';
    let promoted = false;

    // Try to read trust level from SKILL.md frontmatter
    if (fs.existsSync(skillMdPath)) {
      const content = fs.readFileSync(skillMdPath, 'utf8');
      const trustMatch = content.match(/trust-level:\s*(\d+)/);
      if (trustMatch) trustLevel = trustMatch[1];
    }

    // Check if promoted
    const promotedPath = path.join(path.dirname(mintDir), '.claude', 'skills', name);
    promoted = fs.existsSync(promotedPath);

    const status = promoted ? g('promoted') : y('staged');
    console.log(`  ${name.padEnd(25)}${String(trustLevel).padEnd(8)}${status}`);
  }

  console.log('');
  console.log(`  ${d('Promote a skill:')} ${c('mint automate promote <name>')}`);
  console.log('');
}

/**
 * Recursively copy a directory.
 */
function copyDirSync(src, dest) {
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDirSync(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
    }
  }
}
