# FAQ

## Is Ada a replacement for Git?

Not in this alpha. Ada is a Git sidecar.

## What does Ada do today?

Ada adds semantic snapshots, semantic diff, local semantic memory, a local dashboard, and built-in eval scenarios.

## What languages work?

Go and TypeScript/JavaScript.

## Does Ada have a GUI?

Yes. Run:

```bash
ada ui --open
```

## Why does `ada sync` require a clean working tree?

The alpha ties semantic snapshots to committed Git state. That keeps the model predictable while the sidecar architecture is still hardening.

## Is the remote control plane ready?

No. The code exists, but it is experimental and not part of the supported alpha workflow.

## Can I use Ada in CI?

Yes for local CLI-style checks and evals. The initial public-alpha CI focus is on smoke testing, release validation, and reproducible installs.

## What should I try first?

- `ada diff --semantic`
- `ada ui --open`
- `ada eval`
