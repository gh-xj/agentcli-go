---
name: loop-governance
description: Define a simple, repeatable loop governance protocol for onboarding, CI, and iterative quality improvements.
version: 1.0
---

# loop-governance Skill

## Why this skill exists

It turns `agentops loop` into a low-friction project standard: one discoverable profile entrypoint, predictable quality gates, and repeatable failure analysis.

## In scope

- Define the loop protocol for this repo.
- Route teams to one of two policy profiles (`quality`, `lean`).
- Provide the case-study adoption sequence.

## Core protocol

1. Confirm setup and baseline:
   - `agentops loop doctor --repo-root .`
2. Run default gate:
   - `agentops loop quality --repo-root .`
3. Surface policy:
   - `agentops loop profiles --repo-root .`
4. Run behavior regression gate:
   - `agentops loop regression --repo-root .`
5. Investigate failures:
   - `agentops loop lab replay --repo-root . --run-id <run-id> --iter 1`
   - `agentops loop lab compare --repo-root . --run-a <run-id-a> --run-b <run-id-b>`

## Policy source of truth

This repository stores policy in `configs/loop-profiles.json`.

- `quality`: strict team/CI policy with higher score threshold.
- `lean`: lighter policy for quicker local checks (no custom role config by default).

Avoid changing policy flags across scripts; update the profile JSON instead.

## Use flow

| Goal | Command |
| --- | --- |
| Show active policy | `agentops loop profiles --repo-root .` |
| Execute strict gate | `agentops loop quality --repo-root .` |
| Execute local low-noise checks | `agentops loop lean --repo-root .` |
| Execute behavior regression gate | `agentops loop regression --repo-root .` |
| Refresh regression baseline after intentional behavior change | `agentops loop regression --repo-root . --write-baseline` |
| Investigate regressions | `agentops loop lab replay ...` / `agentops loop lab compare ...` |
| Run policy experiment | `agentops loop lab run --repo-root . --mode committee --role-config <path> --max-iterations 1` |

## Case study

- [`case-study.md`](./case-study.md)

## Out of scope

- This skill defines policy, not the internals of loop scoring engines.
- It does not replace `skills/verification-loop/SKILL.md`, which contains command details and investigation defaults.
