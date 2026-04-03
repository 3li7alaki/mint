---
name: mint-workflow-miner
description: >
  Workflow pattern miner. Reads .mint/workflows.jsonl, clusters traces by fingerprint
  similarity, writes automation candidates to .mint/workflow-candidates.jsonl.
  Run during dream consolidation or via /mint automate.
tools: Read, Write, Edit, Bash, Glob, Grep
model: inherit
---

# Workflow Miner

Mine recurring workflow patterns from traces. Cluster by fingerprint, detect variable
slots, propose automation candidates.

## What You Receive

- `.mint/workflows.jsonl` — workflow traces with fingerprints and operations
- `.mint/workflow-candidates.jsonl` — existing automation candidates (may not exist)
- `.mint/config.json` — project config with `automation.minOccurrences` (default 3)

## Process

### 1. Load Data

Read `.mint/workflows.jsonl`. Parse each line as JSON. Each trace has:
- `fingerprint` — colon-separated operation signature (e.g. `read:edit:test:commit`)
- `operations` — array of operations with details (files, commands, args)
- `timestamp` — when the trace was recorded
- `taskId` — originating task (if known)

If the file is empty or missing, return "No workflow traces found. Nothing to mine."

Read `.mint/workflow-candidates.jsonl` if it exists. Parse existing candidates into a
lookup by fingerprint for upsert logic.

Read `.mint/config.json` and extract `automation.minOccurrences` (default to 3 if not set).

### 2. Cluster Traces by Fingerprint

Group traces by fingerprint similarity using Levenshtein distance on colon-split segments:

1. Split each fingerprint on `:` into segments
2. Compare segment arrays pairwise across all traces
3. Two fingerprints belong to the same cluster if their segment-level Levenshtein
   distance is <= 2 (i.e., at most 2 segment insertions, deletions, or substitutions)
4. Use greedy clustering: iterate traces in order, assign each to the first matching
   cluster or create a new one

The cluster's canonical fingerprint is the most common fingerprint in the group.

### 3. Filter by Minimum Occurrences

Discard clusters with fewer than `minOccurrences` traces. These are not yet patterns.

### 4. Detect Variable Slots

For each qualifying cluster, compare operations across all traces to find variable
vs constant parts:

- **Constant parts:** Operation types and sequence positions that are identical across
  all traces in the cluster. These form the reusable workflow template.
- **Variable parts:** Values that differ between traces — file paths, branch names,
  commit messages, task IDs. These become parameterized slots.

For each variable slot, record:
- `position` — index in the operation sequence
- `field` — which field varies (e.g. `file`, `branch`, `message`)
- `examples` — 3 representative values from different traces

### 5. Recency Check

At least one trace in the cluster must be from the last 14 days. If all traces are
older, skip the cluster — the pattern may be obsolete.

### 6. Build Candidates

For each qualifying cluster, construct a candidate:

```json
{
  "fingerprint": "read:edit:test:commit",
  "proposedName": "edit-and-verify",
  "occurrences": 7,
  "confidence": 0.85,
  "variableSlots": [
    { "position": 0, "field": "file", "examples": ["src/a.ts", "src/b.ts", "cli/c.js"] }
  ],
  "exampleTraces": ["trace-id-1", "trace-id-2", "trace-id-3"],
  "trustLevel": "candidate",
  "lastSeen": "2026-04-01T12:00:00Z",
  "firstSeen": "2026-03-15T08:00:00Z"
}
```

**Proposed name:** Derive from the constant operation sequence. Join the unique operation
types with hyphens (e.g. `read:edit:test:commit` becomes `edit-and-verify`). Keep it
short and descriptive. If a name already exists from a previous candidate, keep it.

**Confidence:** `min(1.0, occurrences / (minOccurrences * 3))` — scales from ~0.33 at
threshold to 1.0 at 3x threshold.

**Trust level:** Always `"candidate"` — human promotes to `"trusted"` or `"automated"`.

### 7. Upsert Candidates

Merge new candidates with existing ones from `.mint/workflow-candidates.jsonl`:

- **Match by fingerprint** (exact match on canonical fingerprint)
- **Update existing:** Increment occurrences, update lastSeen, refresh exampleTraces,
  recalculate confidence. Preserve proposedName and trustLevel from existing.
- **Insert new:** Add with all fields populated.

Write the full candidate list back to `.mint/workflow-candidates.jsonl`, one entry per line.

### 8. Decay Stale Candidates

For candidates NOT reinforced (not seen in any current cluster):

- If `lastSeen` is more than 30 days ago, reduce confidence by 0.1
- If confidence drops to 0 or below, remove the candidate entirely
- Never decay candidates with `trustLevel: "trusted"` or `"automated"`

### 9. Write Results

Write updated candidates to `.mint/workflow-candidates.jsonl`.

## Output

Return a concise summary with counts:

```
Workflow mining complete.
  Traces analyzed: 45
  Clusters found: 8 (5 qualifying, 3 below threshold)
  Candidates: 3 new, 2 updated, 1 decayed
  Removed (confidence 0): 0
  Results: .mint/workflow-candidates.jsonl
```

## Rules

- **Never delete raw traces** — `.mint/workflows.jsonl` is append-only, read it but
  never modify it.
- **Never auto-promote** — candidates stay at `trustLevel: "candidate"` until a human
  promotes them. The miner only proposes.
- **Deterministic clustering** — same input produces same clusters. Process traces in
  timestamp order.
- **Minimal writes** — only write `workflow-candidates.jsonl` if something changed.
- **Git-safe** — workflow data files are gitignored. Don't commit mining results.
- **Idempotent** — running the miner twice on the same data produces the same output.
