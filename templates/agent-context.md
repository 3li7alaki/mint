# Agent Context Templates

Defines the exact dynamic prompt structure for each agent. The orchestrator constructs
this from pipeline state and passes it as the `prompt` parameter to the Agent tool.

**How prompt caching works:**
- The agent `.md` file (e.g., `agents/planner.md`) = **static system prompt** — cached by
  Anthropic's API across identical requests. Multiple specs in a wave share this cache.
- The context template below = **dynamic user prompt** — changes per dispatch. This is
  the `prompt` parameter passed to the Agent tool. Keep it minimal.

**Rule: Never duplicate agent instructions in the prompt.** The agent already has its
`.md` file as system prompt. The prompt should contain ONLY the dynamic inputs listed below.

---

## Planner (implement)

```xml
<spec>
{spec XML — full contents of the .xml file}
</spec>

<config>
  <autocommit>{true|false}</autocommit>
  <tdd>{true|false}</tdd>
  <gate-tiers>{true|false — from config.gates.tiered, default true}</gate-tiers>
</config>

<!-- Only on retry (attempt 2+) -->
<retry>
  <attempt>{N}</attempt>
  <previous-failure>{root cause category}: {description}</previous-failure>
  <fix-instruction>{what to do differently}</fix-instruction>
</retry>

<!-- Only on resume after user correction -->
<correction>{user's correction text}</correction>
```

## Planner (fix-blockings)

```xml
<spec>
{spec XML — full contents}
</spec>

<blocking-issues>
  <issue reviewer="{reviewer-name}" severity="BLOCKING" file="{path}" line="{N}">
    {description of the issue}
  </issue>
  <!-- repeat for each blocking issue -->
</blocking-issues>

<config>
  <autocommit>{true|false}</autocommit>
</config>
```

## Decomposer

```xml
<feature>
{feature description from user}
</feature>

<config>
{.mint/config.json contents — or relevant subset}
</config>

<hard-blocks>
{.mint/hard-blocks.md contents}
</hard-blocks>

<learning>
  <issues>
    {relevant entries from .mint/issues.jsonl}
  </issues>
  <wins>
    {relevant entries from .mint/wins.jsonl}
  </wins>
  <instincts>
    {entries from .mint/instincts.jsonl with confidence >= 3}
  </instincts>
</learning>
```

## Spec Reviewer (Stage 1)

```xml
<spec>
{spec XML — full contents}
</spec>

<diff>
{git diff output of what was implemented}
</diff>
```

## Stage 2 Reviewers (quality, security, conventions, tests, performance, business)

All stage 2 reviewers receive the same context structure:

```xml
<diff>
{git diff output of the changes}
</diff>

<files>
{list of modified file paths with brief descriptions}
</files>

<!-- Only for conventions-enforcer, if convention docs exist -->
<conventions>
{contents of docs/conventions.md or relevant convention files}
</conventions>

<!-- Only for business-reviewer, if business docs exist -->
<business-context>
{contents of relevant PRD/BRD/spec docs}
</business-context>
```

## De-sloppifier

```xml
<diff>
{git diff of current changes}
</diff>

<spec>
{spec XML — for understanding intent}
</spec>

<gates>
{gate commands from .mint/config.json — lint, types, tests}
</gates>
```

## Documenter

```xml
<doc>
  <path>{file path to update or create}</path>
  <description>{purpose of this doc}</description>
  <mode>{update|template}</mode>
  <trigger>{on-task-complete|on-architectural-change|manual}</trigger>
</doc>

<change-summary>
{what changed in the code — from planner output or commit message}
</change-summary>

<!-- When manifest sections match -->
<manifest-sections>
  <section id="{id}" heading="{heading}" staleness="{type}">
    <tracks>{glob patterns}</tracks>
    <description>{what this section should contain}</description>
  </section>
</manifest-sections>

<!-- Template mode only -->
<template>{inline template or path}</template>
```

## Researcher

```xml
<question>
{research question from user}
</question>

<config>
{relevant config subset}
</config>
```

## Shipper

```xml
<ship-plan>
{confirmed ship plan with phases and batches}
</ship-plan>

<config>
{.mint/config.json contents}
</config>

<hard-blocks>
{.mint/hard-blocks.md contents}
</hard-blocks>
```

## Verifier (Layer 2)

```xml
<failing-gates>
{output from failed gate commands in layer 1}
</failing-gates>

<config>
{.mint/config.json — gates section}
</config>
```

## Build Error Resolver

```xml
<errors>
{build/type error output}
</errors>

<config>
{.mint/config.json — gates section}
</config>

<scope>
{files in scope for this fix — from spec or orchestrator judgment}
</scope>
```

## Dream Consolidator

```xml
<learning-files>
  <issues path=".mint/issues.jsonl">{entry count}</issues>
  <wins path=".mint/wins.jsonl">{entry count}</wins>
  <instincts path=".mint/instincts.jsonl">{entry count}</instincts>
  <metrics path=".mint/metrics.jsonl">{entry count}</metrics>
  <patterns path=".mint/patterns.jsonl">{entry count}</patterns>
</learning-files>

<config>
{.mint/config.json — relevant subset}
</config>

<!-- Previous dream report if exists -->
<previous-dream>
{.mint/dream-report.md contents, or "none"}
</previous-dream>
```

---

## Cache Efficiency Notes

When a wave runs 4 specs in parallel, each spawns a `mint-planner` agent:
- **Cached (shared):** `agents/planner.md` system prompt (~60 lines, ~2000 tokens)
- **Dynamic (per-spec):** Context template above (~200-500 tokens per spec)
- **Savings:** 4 specs × 2000 tokens static = 8000 tokens. With caching, pay full price
  once (2000) + cache read 3× (~500). Net saving: ~5500 tokens per wave.

Same applies to stage 2 reviewers dispatched in parallel — 5 reviewers share the same
static prompt for their type, each gets a different dynamic context.
