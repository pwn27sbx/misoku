#!/bin/sh
# Builds a throwaway sample workspace for the README demo (docs/demo.tape).
#
#   sh scripts/demo-workspace.sh <empty-or-new-folder>
#
# The notes are invented: the demo never shows a real vault.
set -eu

if [ $# -ne 1 ]; then
	echo "usage: $0 <folder>" >&2
	exit 2
fi
dir=$1
if [ -e "$dir" ] && [ -n "$(ls -A "$dir")" ]; then
	echo "$dir is not empty; refusing to write sample notes into it" >&2
	exit 1
fi
mkdir -p "$dir/Projects" "$dir/Learning"

cat >"$dir/Welcome.md" <<'EOF'
# Welcome to Misoku

A notebook made of **blocks**, saved as plain Markdown.

## Today

- [x] Sketch the trip itinerary
- [ ] Review the [[Docker]] notes
- [ ] Read about [[Kubernetes#Pods|pods]]

## Quick command

![[Docker#^run-nginx]]

> [!tip] Press `?` to see every shortcut.
EOF

cat >"$dir/Learning/Docker.md" <<'EOF'
# Docker

Containers package an app with everything it needs.

## Run a container

Start a web server on port 8080 with `docker run -d -p 8080:80 nginx`. ^run-nginx

```sh
docker ps          # what is running
docker logs web    # what it printed
```

## Images vs containers

| Image | Container |
| --- | --- |
| Blueprint | Running instance |
| Read-only | Has a writable layer |

See also [[Kubernetes]].
EOF

cat >"$dir/Learning/Kubernetes.md" <<'EOF'
# Kubernetes

Runs containers from [[Docker]] across many machines.

## Pods

The smallest unit you deploy: one or more containers that share a network.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
spec:
  containers:
    - name: nginx
      image: nginx
```
EOF

cat >"$dir/Projects/Trip to Kyoto.md" <<'EOF'
# Trip to Kyoto

## Itinerary

1. Fushimi Inari at sunrise
2. Arashiyama bamboo grove
3. Nishiki market for lunch

## Budget

- Train pass: ¥29,650
- Hostel: ¥4,500 / night
EOF

cat >"$dir/Projects/Reading list.md" <<'EOF'
# Reading list

- [ ] The Pragmatic Programmer
- [x] Designing Data-Intensive Applications
- [ ] A Philosophy of Software Design
EOF
