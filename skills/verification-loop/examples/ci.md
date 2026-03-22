# CI Commands

Use for strict quality gates in CI or scheduled workflows.

```bash
task ci
agentops loop doctor --repo-root .
agentops loop quality --repo-root .
agentops loop regression --repo-root .
agentops loop judge --repo-root . --threshold 9.0 --max-iterations 1
go run ./internal/tools/loopbench --mode run --repo-root . --output .docs/onboarding-loop/benchmarks/latest.json
go run ./internal/tools/loopbench --mode check --output .docs/onboarding-loop/benchmarks/latest.json --baseline testdata/benchmarks/loop-benchmark-baseline.json
```
