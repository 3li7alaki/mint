# Workspace Context — Reference

Opt-in cross-repo awareness. Only active if `config.workspace.repos` is defined.

---

## Setup

Read `workspace.repos` array once. For each: name, path, stack, role, dependsOn.
Build lightweight workspace map (names, roles, dependency edges).

## Agent Context Scoping

| Agent | Sees |
|-------|------|
| Planner | Full workspace map |
| Researcher | Full workspace map |
| Spec reviewer | Current repo + dependsOn repos |
| Stage 2 reviewers | Current repo only |
| Documenter | Current repo only |
| Shipper | Full workspace map |

## Cross-Repo Awareness

- Planner notes workspace impact in spec's `<workspace-impact>` field
- Reviewer flags when shared interfaces change (downstream consumers affected)
- Researcher can search dependent repos for patterns
- No cross-repo git operations — awareness only

## Workspace Impact in Summary

If spec has `<workspace-impact>`:
"This change affects: repo-a, repo-b — coordinate before merging."
