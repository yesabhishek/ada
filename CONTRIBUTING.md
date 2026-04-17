# Contributing

Thanks for helping improve Ada.

## Before You Start

- Read the [README](README.md) and [docs/limitations.md](docs/limitations.md).
- Keep in mind that Ada is currently a **Git sidecar**, not a full VCS replacement.
- The supported alpha languages are **Go** and **TypeScript/JavaScript**.

## Local Development

```bash
go test ./...
go build -o bin/ada ./cmd/ada
```

Recommended manual checks:

```bash
./bin/ada eval
./bin/ada doctor
```

## Repo Expectations

- Prefer focused changes over large unrelated refactors.
- Add or update tests when behavior changes.
- Keep user-facing docs in sync with shipped behavior.
- Do not advertise experimental remote/control-plane functionality as production-ready.

## Pull Requests

A strong PR usually includes:

- a clear problem statement
- the behavior change
- test coverage or smoke-test evidence
- docs updates when commands or workflows change

## Reporting Bugs

Please include:

- OS and architecture
- Ada version
- whether you installed from release artifact, Homebrew, install script, or local build
- repro steps
- relevant command output

## Release Discipline

Anything added to the public README or quickstart should be runnable by a first-time user in under 5 minutes on macOS or Linux.
