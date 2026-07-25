# gistsync

Sync hand-picked single files — `.vimrc`, `.gitconfig`, a shell profile — between your
machines, using one secret GitHub Gist per file. Transfers are explicit: you say `push` or
`pull`, one file at a time, and the tool tells you when the two sides have diverged instead
of guessing.

**What it is not:** not a folder sync, not a background daemon, not a merge tool. It never
watches files, never syncs on its own, never merges, and never deletes a gist.

## Prerequisites

The [GitHub CLI](https://cli.github.com) is the only route to GitHub — gistsync holds no
token of its own. Install `gh`, then:

```bash
gh auth login
```

The `gist` scope is required.

## Build

```bash
go build -o gistsync .
```

Cross-compile for the other machine (no CGO, single static binary):

```bash
GOOS=windows GOARCH=amd64 go build -o gistsync.exe .
```

```bash
GOOS=darwin GOARCH=arm64 go build -o gistsync .
```

Put the binary anywhere on your `PATH`.

## First device

```bash
gistsync init
```

```bash
gistsync add ~/.vimrc
```

The logical name comes from the basename with leading dots stripped, so `~/.vimrc` is
tracked as `vimrc`. Pass `--name` to choose your own; you will need it if two files share a
basename.

```bash
gistsync push vimrc
```

That creates the secret gist and prints its URL. The gist's description is `gistsync:vimrc`
— that string, not the ID, is what the other machine looks for.

## Second device

The name is the bridge; the path differs on every machine, so you give it once to `link`.

```bash
gistsync init
```

```bash
gistsync list --remote
```

That shows every gistsync gist on your account and which are tracked here — useful when you
have forgotten what you named things.

```bash
gistsync link vimrc C:/Users/me/_vimrc
```

`link` finds the gist by its description, writes the `[[file]]` entry into this device's
`config.toml`, and records the gist ID. No gist ID is ever copied between machines by hand.
The path may point at a file that does not exist yet — that is the usual case.

```bash
gistsync pull vimrc
```

`pull` writes the file, creating parent directories if needed. `link` never touches your
files.

If the name is already in `config.toml`, the path is optional: `gistsync link vimrc` just
records the gist ID.

### When the file is already there

`link` compares the two copies and tells you where you stand:

- **identical** — it records the sync markers and you are `clean` immediately, no transfer.
- **different** — it links anyway and prints both locations. Nothing has been overwritten,
  and the state is `conflict`: the two copies have no version in common, so `push` and
  `pull` both refuse until you pick a winner with `--force`.

## Everyday use

```bash
gistsync status
```

| state             | meaning                                            | what to do        |
|-------------------|----------------------------------------------------|-------------------|
| `clean`           | both sides match the last sync                     | nothing           |
| `ahead`           | you edited locally                                 | `push`            |
| `behind`          | the other machine pushed                           | `pull`            |
| `conflict`        | both changed since the last sync, or the two copies never shared one | compare, then force one way |
| `never pushed`    | no gist yet for this name                          | `push` or `link`  |
| `missing locally` | the configured path does not exist                 | `pull`            |

Offline, `status` still works: it prints `remote unknown` per file and tells you whether the
local copy changed. `push`, `pull`, `link`, and `add` need the network and fail loudly
without it, changing nothing.

`push` refuses when you are `behind` or in `conflict`; `pull` refuses when you are `ahead` or
in `conflict`. There is no merge — you pick a winner:

```bash
gistsync push --force vimrc
```

```bash
gistsync pull --force vimrc
```

The first overwrites the gist with the local file, the second overwrites the local file with
the gist. Look at both copies before you run either.

To stop tracking a file here:

```bash
gistsync rm vimrc
```

That untracks it locally and leaves both the file on disk and the gist on GitHub alone —
delete the gist yourself on github.com if you want it gone.

## Where things live

Both files sit in `~/.gistsync/` (`%USERPROFILE%\.gistsync\` on Windows):

- **`config.toml`** — one `[[file]]` block per tracked file, with the name and this
  machine's absolute path. `add` and `link` write it for you; editing it by hand is
  supported but never required.
- **`state.json`** — the tool's. Gist IDs and last-sync fingerprints, per device. Editing it
  by hand is how you get a wrong answer from `status`.

Content moves byte-exact in both directions: CRLF stays CRLF, a missing trailing newline
stays missing. Text only — binary and non-UTF-8 files are refused rather than mangled.
