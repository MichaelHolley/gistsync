# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o gistsync .
```

```bash
go vet ./... && gofmt -l .
```

## Architecture

A single-binary Go CLI that syncs individual text files between machines, one secret GitHub Gist per file. No daemon, no merge, no folder sync.

- **Root package `main`** — one file per command (`add.go`, `link.go`, `push.go`, `pull.go`, `status.go`, `list.go`, `rm.go`), dispatched from `run()` in `main.go`. `transfer.go` holds the shared `trackedFile` lookup and the `refuse` error text used by both transfer commands.
- **`internal/store`** — resolves `~/.gistsync/`, creates it, and provides `WriteFileAtomic` (temp file + rename) used for every write, including local file writes in `pull`.
- **`internal/config`** — `config.toml`, written by `add` and `link` (hand-editable, never required). Maps a logical `name` to this machine's absolute `path`. Never synced between devices.
- **`internal/state`** — `state.json`, tool-managed. Per name: `gist_id`, `last_synced_hash` (sha256 of local content), `last_synced_gist_sha` (gist commit SHA).
- **`internal/gh`** — the only route to GitHub, shelling out to the `gh` CLI. gistsync holds no token. Every networked command calls `gh.Preflight()` first.

### Two invariants worth knowing before editing

1. **Name, not gist ID, is the cross-device bridge.** Gists are found by description `gistsync:<name>` (`gh.Description`). Users never copy IDs between machines; `link` resolves the ID locally.
2. **State classification drives every refusal.** `classify()` in `status.go` compares both fingerprints against `state.json` to produce clean/ahead/behind/conflict. `push` refuses on behind/conflict, `pull` on ahead/conflict, unless `--force`. `link` records sync markers *only* when the local file already matches the gist byte-for-byte — inventing them otherwise would make the next `pull` think the sides already match. Empty markers therefore mean "never synced here", which `refuse()` reports as a conflict with no common version rather than as drift.

Content moves byte-exact; `checkContent` in `add.go` rejects NUL bytes and invalid UTF-8 rather than mangling them.

## Docs

Update `README.md` and `CLAUDE.md` whenever behaviour changes — new command, new flag, changed state semantics, changed file layout. It is the only user-facing documentation.
