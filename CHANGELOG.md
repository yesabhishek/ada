# Changelog

All notable changes to Ada will be documented in this file.

The format is based on Keep a Changelog, and versioning is intended to follow Semantic Versioning once Ada reaches stable release status.

## [v0.1.0-alpha.2] - 2026-04-17

### Fixed

- aligned the public module path and release metadata to `github.com/yesabhishek/ada`
- fixed the install-script CI job so the local test server and installer run in the same workflow step
- replaced the tag release workflow with native-runner multi-arch builds so Tree-sitter-backed binaries publish correctly for macOS and Linux
- added automated GitHub release asset aggregation and Homebrew tap formula updates

### Notes

- `v0.1.0-alpha.2` is the first intended public release tag
- `v0.1.0-alpha.1` was used to validate the initial publishing pipeline and exposed cross-platform release issues that are fixed here

## [v0.1.0-alpha.1] - 2026-04-17

### Added

- Go-based local CLI for semantic sidecar workflows
- local `.ada/` SQLite storage for snapshots, symbols, sync runs, tasks, and proposals
- semantic diff and bounded semantic merge for Go and TypeScript/JavaScript
- local read-only UI for snapshots, semantic drift, and sync health
- built-in `ada eval` command for Git-vs-Ada merge comparisons
- public-alpha release scaffolding, docs, installer script, and CI/release automation

### Notes

- This is a public alpha release.
- Ada is a Git sidecar today, not a Git replacement.
- Remote/control-plane features remain experimental.
