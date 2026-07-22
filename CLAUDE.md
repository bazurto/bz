# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Bazurto (bz)

A dependency management and environment modification CLI tool written in Go. Users define dependencies in `.bz.hcl` files, and `bz` resolves versions from GitHub releases, downloads/caches them, and executes commands with a modified environment (PATH, env vars, aliases). Think of it as a cross between `direnv` and `asdf` with GitHub-based dependency distribution.

## Build & Test Commands

```bash
make build          # Build bz binary (with debug symbols)
make test           # Run go vet, deadcode, nilaway, nilness, then go test ./...
make install        # Install to GOPATH
make dist           # Cross-compile for linux/darwin/windows (amd64/arm64)
make grpc           # Regenerate gRPC code from bazurto.proto
make clean          # Remove all build artifacts
go test ./...       # Run just unit tests (skip static analysis)
go test ./lib/...   # Run tests for a specific package
```

The `make test` target requires tools installed via `.requirements`: `deadcode`, `nilaway`, `nilness`, `protoc-gen-go`, `protoc-gen-go-grpc`. These are auto-installed on first run.

## Architecture

### Flow: `main.go` → Engine → Resolvers

1. **main.go** — Parses flags (`--help`, `--version`, `--debug`), creates `AppContext`, registers resolvers, calls `engine.ContextFromConfigDir()` to resolve dependencies, then `engine.Execute()` to run the user's command.

2. **lib/engine.go** (`Engine`) — Core orchestrator. `ContextFromConfigDir` reads `.bz.hcl` (fuzzy config), resolves each dependency through resolvers, downloads/extracts them, writes `.bz.lock`, and returns a `DependencyTree`. `Execute` flattens the tree into an `ExecContext`, sets env vars, resolves aliases, and runs the command.

3. **lib/resolver/** — Three resolver implementations behind the `Resolver` interface:
   - `GithubResolver` — Queries GitHub releases API, matches versions by pattern, downloads OS/arch-specific assets
   - `BazurtoResolver` — Uses gRPC to resolve from a custom package server (proto in `bazurto.proto`)
   - `LocalDevResolver` — Handles `file://` URLs for local development

   Each resolver implements: `ResolveCoord(FuzzyCoord) → LockedCoord` and `DownloadResolvedCoord(LockedCoord) → (extractDir, error, resolved)`.

### Key Model Types (lib/model/)

- **FuzzyCoord** / **FuzzyConfigContent** — User-specified deps with partial versions (e.g., `github.com/bazurto/python#3`)
- **LockedCoord** / **LockedConfigContent** — Resolved deps with exact versions, stored in `.bz.lock`
- **Version** — Custom semver with pattern matching (`3` matches any `3.x.x`, `3.11` matches `3.11.x`)
- **DependencyTree** — Tree of resolved deps; `Resolve()` flattens to `ExecContext`
- **ExecContext** — Final env vars, PATH entries, and aliases for command execution

### Supporting Packages

- **lib/utils/** — Archive extraction (zip/tgz with ZipSlip protection), file operations, circular dependency detection, progress bars, shell expansion via `mvdan.cc/sh`
- **lib/luautils/** — Lua scripting runtime for install/pre-run trigger scripts (uses `gopher-lua`)
- **lib/exec/** — Command execution helpers
- **grpc/bazurto/** — Generated gRPC/protobuf code (regenerate with `make grpc`)

## Conventions

- License: GPL-3.0-only with SPDX headers on all source files
- Config files searched: `.bz.hcl`, `.bz.json`, `.bz` (walks up directory tree)
- Lock file: `.bz.lock` (JSON format, auto-generated, should be committed)
- Cache directory: `~/.bz/cache/deps/`
- Debug logging: `DEBUG=1` env var or `--debug` flag
- Version injected at build time via `-ldflags "-X main.buildInfo=..."` using `.github/revision_get.sh`
- Go 1.24, module path: `github.com/bazurto/bz`
