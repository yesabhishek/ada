# Limitations

Ada `v0.1.0-alpha.2` is intentionally limited.

## Product Limits

- Ada is **not** a full replacement for Git yet.
- Git still owns commits, branches, merge history, and remotes.
- The local CLI and local UI are the supported product surface for this alpha.
- Remote/control-plane code is experimental.

## Platform Limits

- Supported: macOS and Linux
- Not yet supported as a public-alpha target: Windows

## Language Limits

- Supported: Go, TypeScript, JavaScript, TSX, JSX
- Not yet supported: Python and other languages

## Workflow Limits

- `ada sync` requires a clean Git working tree
- the UI is read-only
- semantic merge behavior is intentionally conservative
- Ada may choose conflict over unsafe merge when confidence is low

## Packaging Limits

- Homebrew and install-script paths are part of the alpha release plan, but local `go build` remains the simplest path until the first public tag exists
- container images and Linux package-manager installs are deferred
