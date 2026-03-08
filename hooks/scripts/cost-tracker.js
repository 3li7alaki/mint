#!/usr/bin/env node
/**
 * mint hook: Cost tracker
 * Trigger: Stop
 * Appends session usage metrics to ~/.claude/metrics/costs.jsonl
 */
'use strict';

const path = require('path');
const fs = require('fs');
const os = require('os');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

function estimateCost(model, inputTokens, outputTokens) {
  const rates = { haiku: { in: 0.8, out: 4 }, sonnet: { in: 3, out: 15 }, opus: { in: 15, out: 75 } };
  const m = String(model || '').toLowerCase();
  const r = m.includes('haiku') ? rates.haiku : m.includes('opus') ? rates.opus : rates.sonnet;
  return Math.round(((inputTokens / 1e6) * r.in + (outputTokens / 1e6) * r.out) * 1e6) / 1e6;
}

process.stdin.on('end', () => {
  try {
    const input = data.trim() ? JSON.parse(data) : {};
    const usage = input.usage || input.token_usage || {};
    const inputTokens = Number(usage.input_tokens || usage.prompt_tokens || 0) || 0;
    const outputTokens = Number(usage.output_tokens || usage.completion_tokens || 0) || 0;
    const model = String(input.model || process.env.CLAUDE_MODEL || 'unknown');
    const sessionId = String(process.env.CLAUDE_SESSION_ID || 'default');

    const metricsDir = path.join(os.homedir(), '.claude', 'metrics');
    fs.mkdirSync(metricsDir, { recursive: true });

    const row = {
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      model,
      input_tokens: inputTokens,
      output_tokens: outputTokens,
      estimated_cost_usd: estimateCost(model, inputTokens, outputTokens)
    };
    fs.appendFileSync(path.join(metricsDir, 'costs.jsonl'), JSON.stringify(row) + '\n');
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
