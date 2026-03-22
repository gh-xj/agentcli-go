# agentops

Resource-based agent operations toolkit and scaffold CLI.

## Module

`github.com/gh-xj/agentops` — import as `"github.com/gh-xj/agentops"`

## Architecture Contract

Current architecture is resource-based, not the older DAG `service/operator` layout.

```text
cmd/agentops -> cobrax, resource/*, strategy, dal, internal loop adapters
cobrax       -> resource
resource/*   -> resource, dal, strategy, root contracts
strategy     -> stdlib + yaml
dal          -> stdlib
root package -> shared contracts and report types only
internal/* + tools/harness -> verification / loop infrastructure only
```

Rules:

- Root package files stay as shared contracts, exit codes, and report types.
- `resource/*` must not import `cmd/`, `cobrax/`, or internal harness packages.
- `cobrax/` must stay reusable and must not depend on `dal/`, `strategy/`, `cmd/`, or `internal/`.
- `dal/` and `strategy/` are lower-level packages and must not depend on higher layers.
- The live CLI surface is `agentops`; do not add new `agentcli` references.

## Commands Cheat Sheet

- Show CLI help: `go run ./cmd/agentops --help`
- Run unit tests: `go test ./...`
- Build packages: `go build ./...`
- Run docs drift check: `task docs:check`
- Run canonical CI contract: `task ci`
- Run local aggregate verification: `task verify`
- Inspect loop capabilities: `go run ./cmd/agentops loop --format json capabilities`
- Run behavior regression: `go run ./cmd/agentops loop regression --repo-root .`
- Refresh behavior baseline after intentional loop output changes: `go run ./cmd/agentops loop regression --repo-root . --write-baseline`
- Install local hook: `task hooks:install`

## Documentation Routing

- `README.md`: customer-facing project description and usage.
- `agents.md`: agent onboarding and operating workflow.
- `CLAUDE.md`: durable repo rules and architecture contract.
- `skills/*/SKILL.md`: skill-specific command and workflow guidance.

## Non-Negotiable Rules

- Keep the verification surface aligned with the live `agentops` CLI.
- Do not check in generated root binaries such as `agentops`.
- Keep boundary enforcement in `.golangci.yaml`; prose-only architecture rules are insufficient.
- When behavior changes intentionally, update docs and the regression baseline in the same change.
- Do not put agent-only verification flow in `README.md`; route it to `agents.md` or skill docs.

## Verification Gates

- Before claiming completion, run `task verify`.
- If loop output changed intentionally, run the regression baseline write command and keep `testdata/regression/loop-quality.behavior-baseline.json` in sync.
- `task verify` is the local aggregate gate; `task ci` is the CI contract.
