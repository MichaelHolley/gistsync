# gistsync — Specification

A CLI that syncs individual files across devices using GitHub Gists (one gist per file), driven by explicit `push`/`pull`.

## 1. Goals & non-goals
- **Goal:** keep hand-picked single files in sync across your Windows + macOS machines with explicit, conflict-safe transfers.
- **Non-goals:** directory/tree sync, automatic/background sync, file watching, mobile, binary files, merging.

## 2. Platforms & runtime
- **OS:** Windows and macOS (Linux likely works for free; untested/unsupported).
- **Language:** Go — single static binary per OS, cross-compiled. No runtime dependency beyond `gh`.
- **Hard dependency:** GitHub CLI (`gh`) present and authenticated (`gh auth status`) on every device. All GitHub access shells out to `gh` (`gh gist ...`, `gh api ...`).

## 3. Storage model
- **One gist per tracked file.** Gists are **secret** (unlisted).
- **Cross-device key = gist description** of the exact form `gistsync:<name>`.
  - `<name>` is the logical tracking name, unique across your tracked set.
  - The file **inside** the gist is named `<name>` (not the original basename).
- **Discovery:** a device finds a file's gist by listing gists and matching description `gistsync:<name>`. Gist IDs are **not** stored in shared config — they live in per-device local state, resolved via description.

## 4. Local layout (`~/.gistsync/`)
Single dotfolder on both OSes (`$HOME/.gistsync/`, `%USERPROFILE%\.gistsync\`).
- `config.toml` — per-device tracked entries (paths differ per OS; `<name>` is the shared identity).
- `state.json` — per-device sync state, one record per tracked name.

### config.toml (per device, authored per device)
```toml
[[file]]
name = "vimrc"                 # logical name = gist key; unique
path = "C:/Users/me/.vimrc"    # local absolute path (differs per OS)
```

### state.json (managed by tool)
Per name: `{ name, gist_id, last_synced_hash, last_synced_gist_sha }`
- `last_synced_hash` — SHA-256 of local file content at last successful sync.
- `last_synced_gist_sha` — gist commit/version SHA at last successful sync.

## 5. State classification
On a command, compute:
- `local_hash` = hash of file on disk now.
- `remote_sha` = current gist version SHA (from GitHub).
- Compare against `last_synced_hash` / `last_synced_gist_sha`:

| local vs last | remote vs last | state    |
|---------------|----------------|----------|
| same          | same           | clean    |
| changed       | same           | ahead    |
| same          | changed        | behind   |
| changed       | changed        | conflict |

## 6. Conflict policy
- **Refuse + report.** `push`/`pull` that detect `conflict` stop with a clear message; no data touched.
- **Override:** `--force`.
  - `push --force` → local wins, overwrite gist.
  - `pull --force` → remote wins, overwrite local.
- No auto-merge, ever.

## 7. Line endings & encoding
- **Byte-exact.** No CRLF/LF normalization; bytes on disk go up verbatim, bytes in gist come down verbatim. (CRLF↔LF differences count as real changes → may surface as conflicts; accepted tradeoff.)
- **Text-only.** Detect binary (e.g. NUL byte in content); refuse with a clear error. No base64 path.

## 8. Commands (explicit direction only; **no `sync` command**)
All transfer commands require an explicit `<name>` (one file per invocation).

- `init` — create `~/.gistsync/`, empty `config.toml` + `state.json`.
- `add <path> [--name <name>]`
  - Register a file. `<name>` defaults to basename; **basename collision → error, require `--name`**.
  - Creates the gist (description `gistsync:<name>`, secret) on first `push`.
  - **If a gist with description `gistsync:<name>` already exists → error**, directing user to `link`.
- `link <name>` — attach an already-tracked name to its existing remote gist (found by description) on this device; sets `gist_id` in state. Follow with `pull` to fetch.
- `rm <name>` — untrack locally. **Leaves the remote gist untouched** (orphans accepted).
- `status` — per tracked file: clean / ahead / behind / conflict.
  - **Offline:** show local-only info + `remote unknown` note; do not hard-fail.
- `list` — tracked files with `<name>`, local path, gist URL.
- `push <name> [--force]` — upload local → gist. Refuse on conflict unless `--force`.
- `pull <name> [--force]` — download gist → local (creating the file if absent). Refuse on conflict unless `--force`.

## 9. Network behavior
- `push`/`pull` **hard-error** when GitHub is unreachable.
- `status` degrades gracefully offline (local state + `remote unknown`).

## 10. Edge cases (defined)
- **Fresh device pull, file absent locally:** `pull` creates it at the configured path.
- **Basename collision on `add`:** error → require `--name`.
- **Duplicate gist on `add`:** error → use `link`.
- **Name not unique across tracked set:** rejected at `add`.
- **Binary content:** rejected at `add`/`push`.
- **Removed from config / deleted locally:** gist left intact.

## 11. Open items deferred (explicitly out of scope for v1)
- No file watcher, interval, or login-triggered sync.
- No auto-merge / 3-way merge.
- No binary support.
- No gist deletion.
- No shared/remote config distribution (config authored per device; `<name>` is the bridge).
