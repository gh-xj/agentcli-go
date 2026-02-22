# Examples

Runnable example CLIs generated with `agentcli`.

## Included

- `file-sync-cli/`: deterministic local file sync style workflow
- `http-client-cli/`: API-request style command workflow
- `deploy-helper-cli/`: preflight/deploy workflow skeleton

## Verify

Each example is independently runnable and verifiable:

```bash
cd examples/file-sync-cli && task verify
cd examples/http-client-cli && task verify
cd examples/deploy-helper-cli && task verify
```
