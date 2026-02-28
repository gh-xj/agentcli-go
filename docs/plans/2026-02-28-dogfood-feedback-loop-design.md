# Dogfood Feedback Loop Design

- Date: 2026-02-28
- Status: Approved
- Scope: Cross-repo reusable skill with GitHub issue feedback channel

## Problem

When dogfooding coding workflows across products, failures and friction points are discovered, but feedback capture is inconsistent and often lost. We need a reusable skill that captures high-value signals and routes actionable feedback into product repositories with low noise.

## Goals

- Provide one reusable cross-repo dogfooding skill.
- Auto-capture signals from:
  - CI/test failures
  - runtime/command errors
  - docs/skill drift checks
- Route feedback to the correct GitHub repository with inference-first routing.
- Keep issue quality high and noise low through dedupe and confidence gating.
- Preserve a local audit trail for replay and tuning.

## Non-Goals (v1)

- Full incident management workflow beyond issue creation/comment updates.
- Complex multi-system synchronization (e.g., mirrored trackers).
- Fully autonomous routing with no operator override path.

## Chosen Approach

Use a **config-first router with inference fallback** and GitHub as the feedback destination.

Why this approach:

- Supports cross-repo usage with deterministic controls.
- Keeps flexibility through inference while reducing misrouting risk.
- Best fit for the success metric: high-value issues with low noise.

## Success Metric

Primary v1 metric (2-4 week window):

- Balanced quality signal: high-value issues with low noise/spam.

Supporting indicators:

- Duplicate issue rate reduced by fingerprint dedupe.
- Pending-review ratio stays bounded while maintaining high-confidence auto-publish.
- Median time from signal detection to routed issue/comment remains low.

## Architecture

### 1) Signal Collector

Ingests automatic signals from CI/test failures, runtime/command errors, and docs/skill drift checks.

### 2) Normalizer + Scorer

Transforms raw events into canonical records and computes confidence/severity used by routing and publish gates.

### 3) Repo Router (inference-first)

Infers target repo from runtime context (cwd, git remote, touched files, command context), with manual override support.

### 4) Issue Publisher (GitHub)

Creates issues or appends comments for recurring fingerprints. Applies labels and structured template fields.

### 5) Feedback Ledger

Persists local event history and routing outcomes for audit, dedupe windows, and replay.

## Data Model (Canonical Event)

Required fields:

- `event_id`
- `event_type`
- `signal_source`
- `repo_guess`
- `repo_confidence`
- `branch`
- `commit`
- `command`
- `error_summary`
- `evidence_paths`
- `fingerprint`
- `timestamp`

Routing/audit fields:

- `routing_reason`
- `dedupe_status`
- `publish_decision`
- `issue_url` (if published)

## Data Flow

1. Capture signal from supported sources.
2. Normalize into canonical event payload.
3. Score confidence/severity and compute fingerprint.
4. Check dedupe and cooldown against ledger.
5. Resolve repository via inference (or override if provided).
6. Publish action:
   - new fingerprint -> create issue
   - existing open fingerprint -> append comment with fresh evidence
7. Persist ledger record and operator summary.

## Noise Control Strategy

- Fingerprint key uses: repo + signal type + command + stable error signature.
- Cooldown windows suppress repeated bursts.
- Confidence threshold blocks low-confidence auto-publish.
- Ambiguous routing becomes `pending-review` instead of auto-issue.

## Error Handling and Fail-Safes

- Low-confidence routing: do not auto-open issue; store pending-review with next action.
- GitHub transient failures: bounded retries with backoff.
- GitHub hard failures/token issues: queue locally; expose replay path.
- Dedupe guardrails: create vs append-comment decision by fingerprint/open-state.
- Manual escape hatch: `--force-new` to open a new issue intentionally.

## Replay and Recovery

- `dogfood feedback replay` re-attempts queued events after auth/network recovery.
- Deterministic event IDs ensure replay traceability.

## Testing and Verification

### Unit tests

- Fingerprint stability and dedupe logic.
- Routing confidence calculation and threshold gating.
- Cooldown behavior.

### Integration tests

- End-to-end signal -> routing -> issue publish/comment update.
- Ambiguous routing -> pending-review.
- Replay path after simulated API/auth failures.

### Contract tests

- Canonical event schema and ledger schema validation.
- Issue template completeness requirements.

### Noise-quality regression

- Fixture scenarios for repeated/noisy failures.
- Assertions for no duplicate spam and correct append-comment behavior.

### Operational verification

- Integrate with repo gates (`task ci`, `task verify`) where relevant.
- Add dogfood report summary:
  - opened vs deduped vs pending-review
  - confidence distribution
  - sampled false-positive review count

## Rollout Plan

1. Pilot with a small product subset and tune thresholds.
2. Expand to more repos with shared defaults and per-repo overrides.
3. Review weekly quality metrics; adjust dedupe/cooldown/routing thresholds.

## Open Decisions Deferred to Implementation Plan

- Exact confidence scoring formula and initial thresholds.
- Repository mapping file format and precedence rules.
- Standard label taxonomy for created issues.
- CLI command surface details and config file paths.

