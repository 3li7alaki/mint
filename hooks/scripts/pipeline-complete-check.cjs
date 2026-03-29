#!/usr/bin/env node
/**
 * mint hook: Pipeline completion check
 * Trigger: Stop
 *
 * Blocks Claude from stopping if an active mint session has incomplete pipeline steps.
 * Exit code 0 = allow stop. Exit code 2 = block stop (pipeline incomplete).
 *
 * Checks:
 * 1. Is there an active session? (mintInvoked = true)
 * 2. If plan mode: are there pipeline-state.json files with pending steps?
 * 3. If any spec has execution.json without reviews populated, flag it.
 */
'use strict';

const fs = require('fs');
const path = require('path');

// Find project root (walk up looking for .mint/)
function findProjectRoot() {
  let dir = process.cwd();
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, '.mint'))) return dir;
    dir = path.dirname(dir);
  }
  return null;
}

function main() {
  const projectRoot = findProjectRoot();
  if (!projectRoot) process.exit(0); // no mint project, allow stop

  const mintDir = path.join(projectRoot, '.mint');
  const sessionsDir = path.join(mintDir, 'sessions');

  // Find active session (most recent first — IDs are timestamp-prefixed)
  let activeSession = null;
  try {
    const files = fs.readdirSync(sessionsDir)
      .filter(f => f.endsWith('.json'))
      .sort()
      .reverse();

    for (const f of files) {
      try {
        const s = JSON.parse(fs.readFileSync(path.join(sessionsDir, f), 'utf8'));
        if (s.mintInvoked) {
          activeSession = s;
          break;
        }
      } catch { /* skip invalid */ }
    }
  } catch { /* no sessions dir */ }

  if (!activeSession) process.exit(0); // no active session, allow stop

  // Quick mode doesn't have pipeline-state — allow stop
  if (activeSession.mode !== 'plan') process.exit(0);

  // Find task directories with execution tracking
  const tasksDir = path.join(mintDir, 'tasks');
  if (!fs.existsSync(tasksDir)) process.exit(0);

  const issues = [];

  try {
    const taskSlugs = fs.readdirSync(tasksDir).filter(d =>
      fs.statSync(path.join(tasksDir, d)).isDirectory()
    );

    for (const slug of taskSlugs) {
      const taskDir = path.join(tasksDir, slug);

      // Check each spec directory for execution.json
      const specDirs = fs.readdirSync(taskDir).filter(d => {
        const fullPath = path.join(taskDir, d);
        return fs.statSync(fullPath).isDirectory() && fs.existsSync(path.join(fullPath, 'execution.json'));
      });

      for (const specDir of specDirs) {
        const execPath = path.join(taskDir, specDir, 'execution.json');
        try {
          const exec = JSON.parse(fs.readFileSync(execPath, 'utf8'));

          // Skip completed specs
          if (exec.status === 'passed' || exec.status === 'failed') continue;

          // Check for missing reviews (the most common skipped step)
          const gates = exec.gates || {};
          const reviews = exec.reviews || {};
          const hasGates = Object.keys(gates).length > 0;
          const hasReviews = Object.keys(reviews).length > 0;

          if (hasGates && !hasReviews && exec.status !== 'failed') {
            issues.push(`${slug}/${specDir}: gates ran but no reviews — pipeline incomplete`);
          }

          // Check pipeline-state.json if it exists
          const pipelinePath = path.join(taskDir, specDir, 'pipeline-state.json');
          if (fs.existsSync(pipelinePath)) {
            try {
              const pipeline = JSON.parse(fs.readFileSync(pipelinePath, 'utf8'));
              const pending = pipeline.pendingSteps || [];
              if (pending.length > 0) {
                issues.push(`${slug}/${specDir}: ${pending.length} pipeline steps remaining (${pending.join(', ')})`);
              }
            } catch { /* skip invalid */ }
          }
        } catch { /* skip invalid */ }
      }
    }
  } catch { /* tasks dir read error */ }

  if (issues.length > 0) {
    // Output goes to stderr so the user sees it
    console.error(`\n[mint] Pipeline incomplete — ${issues.length} issue(s):\n`);
    for (const issue of issues) {
      console.error(`  - ${issue}`);
    }
    console.error('\nComplete the pipeline steps or delete the session to stop.');
    console.error('To force stop: delete .mint/sessions/*.json\n');
    process.exit(2); // Block stop
  }

  process.exit(0); // All clear
}

main();
