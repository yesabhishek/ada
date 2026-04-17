# Ada

Ada is a **public alpha semantic sidecar for Git**.

It helps AI-heavy engineering teams inspect code semantically instead of only as text. Today, Ada is not a Git replacement. Git still handles commits, branches, merges, and remotes. Ada layers on:

- semantic snapshots in a local `.ada/` SQLite store
- semantic diff for supported languages
- local query and proposal workflows
- a lightweight local dashboard
- Git-vs-Ada eval scenarios

## Alpha Contract

Ada `v0.1.0-alpha.2` is intentionally narrow:

- Ada is a **Git sidecar**, not a full VCS replacement
- supported languages are **Go** and **TypeScript/JavaScript**
- supported platforms are **macOS** and **Linux**
- the **local CLI + local UI** are the supported product surface
- remote/control-plane pieces exist in the repo but are **experimental**

## 5-Minute Quickstart

Build the CLI:

```bash
git clone https://github.com/yesabhishek/ada.git
cd ada
go build -o bin/ada ./cmd/ada
```

Create a small Git repo to try it on:

```bash
mkdir -p /tmp/ada-demo
cd /tmp/ada-demo

git init
git config user.name "Your Name"
git config user.email "you@example.com"

cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a + b
}
EOF

git add .
git commit -m "initial"
```

Start Ada and create the first semantic snapshot:

```bash
/path/to/ada/bin/ada start .
/path/to/ada/bin/ada sync
```

Make a change and inspect it:

```bash
cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a - b
}
EOF

/path/to/ada/bin/ada diff --semantic
/path/to/ada/bin/ada diff --text
/path/to/ada/bin/ada status --sync
/path/to/ada/bin/ada ui --open
```

Commit with Git, then sync Ada again:

```bash
git add .
git commit -m "change add logic"
/path/to/ada/bin/ada sync
```

## Commands

Supported alpha commands:

```bash
ada start .
ada sync
ada status --sync
ada diff --semantic
ada diff --text
ada ask "add"
ada task add "Refactor cache"
ada propose "Semantic merge proposal"
ada resolve "Keep Thread B's logic."
ada rewind search "before auth refactor"
ada rewind apply <snapshot-id> --dry-run
ada ui --open
ada eval
ada eval --format markdown
ada version
ada doctor
```

## Install

Public-alpha distribution paths:

- direct binary download from GitHub Releases
- install script: `curl -fsSL https://raw.githubusercontent.com/yesabhishek/ada/main/scripts/install.sh | bash`
- Homebrew tap for macOS/Linux

Until the first release is tagged, local `go build` is the reliable path.

## What To Try First

- `ada diff --semantic` after a small code edit
- `ada ui --open` to watch snapshots and sync state
- `ada eval` to compare built-in Git vs Ada merge scenarios

## Documentation

- [Quickstart](docs/quickstart.md)
- [Limitations](docs/limitations.md)
- [FAQ](docs/faq.md)
- [Alpha Roadmap](docs/alpha-roadmap.md)
- [Release Notes: v0.1.0-alpha.2](docs/releases/v0.1.0-alpha.2.md)
- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)

## Known Limits

- `ada sync` requires a clean Git working tree
- Ada does not replace `git commit`, `git merge`, `git push`, or `git pull`
- the local UI is read-only
- the remote/control-plane code is experimental and not part of the supported alpha workflow

## License

Apache-2.0. See [LICENSE](LICENSE).
