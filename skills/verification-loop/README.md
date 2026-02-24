# Verification Loop Skill

Front page for project-local loop verification guidance.

## Source of truth

- `skills/verification-loop/SKILL.md`

CLI surface:

- `agentcli loop [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [--format text|json|ndjson] [--summary path] [--no-color] [--dry-run] [--explain] [command flags]`
- `agentcli loop lab [compare|replay|run|judge|autofix] [advanced flags]`

## What this includes

- Baseline quality and doctor checks
- Profile-aware loop gates
- Behavior-regression baseline checks
- Lab workflows for replay and compare diagnostics
- Case-study links for first-time onboarding

## Quick references

- `../loop-governance/SKILL.md`
- `../loop-governance/case-study.md`
- `../agents.md`
- `../README.md` (user-facing usage context)

## Example pages

- `examples/lean.md`
- `examples/lab.md`
- `examples/ci.md`
- `CHECKLIST.md`
