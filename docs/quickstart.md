# Quickstart

This guide gets you from zero to a working local Ada demo in about 5 minutes.

## 1. Build Ada

```bash
git clone https://github.com/yesabhishek/ada.git
cd ada
go build -o bin/ada ./cmd/ada
```

## 2. Create a tiny Git repo

```bash
mkdir -p /tmp/ada-demo
cd /tmp/ada-demo

git init
git config user.name "Your Name"
git config user.email "you@example.com"
```

Create a file:

```bash
cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a + b
}
EOF
```

Commit it with Git:

```bash
git add .
git commit -m "initial"
```

## 3. Start Ada

```bash
/path/to/ada/bin/ada start .
/path/to/ada/bin/ada sync
```

This creates `.ada/` and records the first semantic snapshot.

## 4. Make a change

```bash
cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a - b
}
EOF
```

Inspect it:

```bash
/path/to/ada/bin/ada diff --semantic
/path/to/ada/bin/ada diff --text
/path/to/ada/bin/ada status --sync
```

## 5. Open the UI

```bash
/path/to/ada/bin/ada ui --open
```

Default URL:

```text
http://127.0.0.1:4173
```

## 6. Commit again and sync again

```bash
git add .
git commit -m "change add logic"
/path/to/ada/bin/ada sync
```

## 7. Run the built-in evals

```bash
/path/to/ada/bin/ada eval
```

## Workflow Reminder

Ada currently works **alongside Git**:

- edit code
- inspect with Ada
- commit with Git
- sync with Ada
