# Agent entry point

This repo is a take-home engineering case study. Start with [the command center](docs/COMMAND_CENTER.md) for the problem, decision log, and work history. The original exercise instructions, RFC-0042, and Interest Model v2 were supplied separately and are not included in this submission.

## Read the submission

Read [REVIEW.md](docs/REVIEW.md), [DESIGN.md](docs/DESIGN.md), [the POC README](poc/README.md), and [AI_NOTES.md](docs/AI_NOTES.md). The command center is a work record, not a substitute for those deliverables.

For the code, follow [mock data](poc/config/scenarios.json) to [mock partners](poc/mockpartner/mock_partner.go) to [allocation](poc/allocation/allocator.go) to [measurement](poc/cmd/poc/measurement.go). Judge the POC against its two claimed properties. Cohort ingestion, event pipelines, trained models, and the portal are design proposals, not implemented features.

## Verify the proof of concept

Use Go 1.22 or newer. From `poc/`, run `go test ./...`, `go test -race ./...`, and `go vet ./...`. The POC needs no credentials or external services. Check the README's measured scenarios and preserve its stated limits.
