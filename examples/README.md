# Examples

Runnable example CLIs generated with `agentcli`.

## Included

- `file-sync-cli/`: deterministic local file sync style workflow
- `http-client-cli/`: API-request style command workflow
- `deploy-helper-cli/`: preflight/deploy workflow skeleton

## Cross-repo orchestration

For replay-style orchestration across repositories, scaffold a wrapper command preset:

```bash
agentcli add command --preset task-replay-orchestrator replay-orchestrate
go run . replay-orchestrate --repo ../external-repo --task replay:emit --env IOC_ID=123 --env MODE=baseline --timeout 2m
```

## Verify

Each example is independently runnable and verifiable:

```bash
cd examples/file-sync-cli && task verify
cd examples/http-client-cli && task verify
cd examples/deploy-helper-cli && task verify
```
