# ADR 0004: Narrow hackathon release scope

- Status: accepted
- Date: 2026-09-20

## Context

The final release must demonstrate depth in one real coding-agent workflow. A
scope audit is needed to distinguish implemented platform surfaces from live
evidence still required for release, and to prevent deferred product ideas from
expanding the submission.

## Decision

The submission uses one AWS region, one default sandbox image, one MCP path,
and one verified coding-agent client. The HTTP `/v1` API remains canonical;
the SDK and MCP server are adapters. A feature is not called verified until it
has a recorded local or live check appropriate to its boundary.

## Must-build audit

The PRD must-build list maps to these implementation tickets and evidence:

| PRD capability | Ticket coverage | Release evidence |
| --- | --- | --- |
| AWS sandbox creation and destruction | 21, 24, 25, 27 | ECS task lifecycle and cleanup |
| Command execution and live output | 09, 10, 15 | Command IDs, exit status, streamed output |
| Filesystem operations and repository cloning | 07, 08, 16, 34, 39 | `/workspace` operations and fixture journey |
| Dependency installation and resource timeouts | 09, 19, 35, 41 | Fixture install, timeout case, cleanup |
| Workspace snapshots and restoration using S3 | 17, 18, 20, 27, 39 | S3 object/checksum and restored test |
| Basic dashboard and execution history | 32, 33 | Lifecycle, terminal, and timeline capture |
| TypeScript SDK | 28, 29, 30 | SDK lifecycle repeat against `/v1` |
| Canonical authenticated `/v1` API | 03, 04, 12, 14, 15, 16, 20 | Contract, auth, idempotency, SSE, typed errors |
| One MCP agent-native integration | 36, 37, 38, 39 | Verified client invocation and `journeyStatus: passed` |
| Visible AWS infrastructure | 21, 24, 25, 26, 27, 44 | ECS, CloudWatch, S3, DynamoDB, and no-credential evidence |

The implementation mapping is not a claim that a live deployment has passed.
The final rehearsal in [release-rehearsal.md](../release-rehearsal.md) is the
release gate for live evidence.

## Intentionally deferred

The following PRD items remain out of the hackathon slice:

- GPU environments, browser automation, Kubernetes, and custom images.
- Billing, enterprise authentication, organizations, and marketplace support.
- Multiple regions, advanced networking controls, and complex autoscaling.
- Local product mode, multiple MCP implementations, and provider-specific
  plugins.
- Parallel sandboxes, agent handoff, sandbox templates, and artifact downloads
  unless they are needed to support the single vertical slice.

These are future milestones, not release blockers. Do not add infrastructure or
claims for them to make the final demo appear broader.

## Consequences

The release can be judged from one coherent story: an agent receives a
temporary AWS computer, performs real work, preserves it in S3, resumes in a
new task, and cleans up. The explicit evidence gate makes unavailable AWS or
agent verification visible instead of silently substituting local mocks.
