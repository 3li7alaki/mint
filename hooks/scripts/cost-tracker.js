#!/usr/bin/env node
/**
 * mint hook: Cost tracker
 * Trigger: Stop
 * Reads the session transcript to sum token usage, then appends metrics to ~/.claude/metrics/costs.jsonl
 *
 * Stop hook stdin: { session_id, transcript_path, last_assistant_message, cwd, ... }
 * Actual usage lives in the transcript file (assistant messages with message.usage).
 */
'use strict';

const path = require('path');
const fs = require('fs');
const os = require('os');
const readline = require('readline');

const MAX_STDIN = 1024 * 1024;
let data = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  if (data.length < MAX_STDIN) data += chunk.substring(0, MAX_STDIN - data.length);
});

const RATES = {
  haiku:  { in: 0.80, cache_write: 1.00, cache_read: 0.08, out: 4.00 },
  sonnet: { in: 3.00, cache_write: 3.75, cache_read: 0.30, out: 15.00 },
  opus:   { in: 15.00, cache_write: 18.75, cache_read: 1.50, out: 75.00 },
};

function getRates(model) {
  const m = String(model || '').toLowerCase();
  if (m.includes('haiku')) return RATES.haiku;
  if (m.includes('opus')) return RATES.opus;
  return RATES.sonnet;
}

function estimateCost(rates, tokens) {
  const perM = (count, rate) => (count / 1e6) * rate;
  return Math.round((
    perM(tokens.input, rates.in) +
    perM(tokens.cache_creation, rates.cache_write) +
    perM(tokens.cache_read, rates.cache_read) +
    perM(tokens.output, rates.out)
  ) * 1e6) / 1e6;
}

async function parseTranscript(transcriptPath) {
  const totals = { input: 0, output: 0, cache_creation: 0, cache_read: 0 };
  let model = 'unknown';
  let turnCount = 0;

  const stream = fs.createReadStream(transcriptPath, { encoding: 'utf8' });
  const rl = readline.createInterface({ input: stream, crlfDelay: Infinity });

  for await (const line of rl) {
    if (!line.trim()) continue;
    try {
      const obj = JSON.parse(line);
      const msg = obj.message;
      if (!msg || msg.role !== 'assistant') continue;

      turnCount++;
      if (msg.model) model = msg.model;

      const u = msg.usage;
      if (!u) continue;
      totals.input += Number(u.input_tokens || 0);
      totals.output += Number(u.output_tokens || 0);
      totals.cache_creation += Number(u.cache_creation_input_tokens || 0);
      totals.cache_read += Number(u.cache_read_input_tokens || 0);
    } catch { /* skip malformed lines */ }
  }

  return { model, totals, turnCount };
}

process.stdin.on('end', async () => {
  try {
    const input = data.trim() ? JSON.parse(data) : {};
    const transcriptPath = input.transcript_path;
    const sessionId = input.session_id || 'default';

    if (!transcriptPath || !fs.existsSync(transcriptPath)) {
      process.stdout.write(data);
      return;
    }

    const { model, totals, turnCount } = await parseTranscript(transcriptPath);

    if (turnCount === 0) {
      process.stdout.write(data);
      return;
    }

    const rates = getRates(model);
    const cost = estimateCost(rates, totals);

    const metricsDir = path.join(os.homedir(), '.claude', 'metrics');
    fs.mkdirSync(metricsDir, { recursive: true });

    const row = {
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      model,
      turns: turnCount,
      input_tokens: totals.input,
      output_tokens: totals.output,
      cache_creation_tokens: totals.cache_creation,
      cache_read_tokens: totals.cache_read,
      estimated_cost_usd: cost,
    };
    fs.appendFileSync(path.join(metricsDir, 'costs.jsonl'), JSON.stringify(row) + '\n');
  } catch { /* non-blocking */ }
  process.stdout.write(data);
});
