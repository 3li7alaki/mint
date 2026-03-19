---
description: "Analyze project documentation and build a comprehensive doc-manifest with section tracking"
---

# doc-setup

Build or enrich a project's `.mint/doc-manifest.json` by analyzing existing documentation.

## When to Use

- After `mint init` for an existing project with docs
- When adding doc tracking to a project that already has documentation
- When you want to map existing doc sections to code dependencies

## What It Does

1. **Scan existing docs** — find all markdown files in the project (README, CONTRIBUTING, docs/, etc.)
2. **Parse sections** — read each doc and identify sections by their headings (## level)
3. **Infer tracking** — for each section, infer which code artifacts it depends on:
   - Sections mentioning file paths -> track those paths
   - Sections with tables listing files/agents/commands -> track the corresponding directories
   - Sections about config -> track config files
   - Sections about architecture -> track agent files and SKILL.md
4. **Choose staleness strategy** — based on section content:
   - Tables/lists of files -> `glob-count`
   - Config schemas -> `content-hash`
   - Narrative descriptions -> `git-diff`
5. **Write manifest** — save to `.mint/doc-manifest.json`
6. **Report** — show what was detected and mapped

## Process

```
1. Read existing .mint/doc-manifest.json (if any) as base
2. Scan project for doc files
3. For each doc:
   a. Read the file
   b. Parse ## headings into sections
   c. For each section:
      - Generate a kebab-case ID from the heading
      - Analyze content for file references, directory mentions, config keys
      - Infer tracks[] glob patterns
      - Choose staleness strategy
   d. Add to manifest
4. Merge with existing manifest (don't overwrite user customizations)
5. Write .mint/doc-manifest.json
6. Show summary
```

## Output

```
doc-setup complete

Manifest: .mint/doc-manifest.json
Docs: 5 files, 23 sections tracked

  README.md (5 sections)
    how-it-works      tracks: skills/mint/SKILL.md          staleness: git-diff
    pipeline           tracks: agents/*.md                   staleness: glob-count
    core-features      tracks: skills/mint/SKILL.md          staleness: git-diff
    configuration      tracks: .mint/config.json             staleness: content-hash
    ecosystem          tracks: skills/mint/SKILL.md          staleness: git-diff

  CONTRIBUTING.md (2 sections)
    project-structure  tracks: agents/*.md, commands/*.md     staleness: glob-count
    writing-agents     tracks: agents/*.md                   staleness: glob-count

  ...

Run mint doctor to verify doc health.
```

## Rules

- **Don't overwrite user sections.** If a section already exists in the manifest with tracks, keep it.
- **Infer conservatively.** Only add tracks you're confident about. Empty tracks is better than wrong tracks.
- **Match existing style.** Use kebab-case for section IDs. Keep descriptions concise.
- **This is a setup tool.** Run once to bootstrap, then users customize. Don't auto-run on every task.
