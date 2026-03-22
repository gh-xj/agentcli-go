# Contributing to agentops

Thanks for improving agentops.

## Repository setup

- Install Go (matching your environment’s `go.mod` requirement).
- Install the CLI helper if you need scaffold flows:

```bash
go install github.com/gh-xj/agentops/cmd/agentops@v0.2.1
```

## Core checks

Before opening a PR, run at least:

```bash
go test ./...
task verify
```

Optional deep checks:

```bash
agentops loop doctor --repo-root .
agentops loop quality --repo-root . --threshold 9.0 --max-iterations 1
```

## PR expectations

- Keep API additions small and generic.
- Prefer deterministic behavior and explicit error paths.
- Update docs where behavior changes:
  - `README.md` for public API/use-case changes
  - `agents.md` for agent workflow updates
  - `CLAUDE.md` for durable repo-level rules
  - `docs/documentation-conventions.md` for routing exceptions
- Add/adjust tests for behavior changes.
- If you touch skill commands, update `skills/*/SKILL.md` and keep examples aligned.

## Feedback

When reporting issues, include:

- What command was run
- Input parameters
- Expected vs actual output
- Reproducible steps
